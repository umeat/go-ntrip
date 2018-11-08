package main

import (
    "fmt"
    "net/http"
    "log"
    "context"
    "github.com/satori/go.uuid"
    "sync"
)

type Client struct {
    Id string
    Writer http.ResponseWriter
    Request *http.Request
    Cancel context.CancelFunc
}

type Mountpoint struct {
    sync.RWMutex
    Id string
    Clients map[string]*Client
}

func (mount *Mountpoint) AddClient(client *Client) {
    mount.Lock()
    mount.Clients[client.Id] = client
    mount.Unlock()
}

func (mount *Mountpoint) DeleteClient(id string) {
    mount.Lock()
    delete(mount.Clients, id)
    mount.Unlock()
}

func (mount *Mountpoint) Write(data []byte) {
    mount.Lock()
    for _, client := range mount.Clients {
        fmt.Fprintf(client.Writer, "%s", data)
        client.Writer.(http.Flusher).Flush()
    }
    mount.Unlock()
}

type MountpointCollection struct {
    sync.RWMutex
    mounts map[string]*Mountpoint
}

func (m MountpointCollection) NewMountpoint(id string) (mount *Mountpoint) {
    mount = &Mountpoint{Id: id, Clients: make(map[string]*Client)}
    m.Lock()
    m.mounts[id] = mount
    m.Unlock()
    return mount
}

func (m MountpointCollection) DeleteMountpoint(id string) {
    m.Lock()
    delete(m.mounts, id)
    m.Unlock()
}

func (m MountpointCollection) GetMountpoint(id string) (mount *Mountpoint, ok bool) {
    m.RLock()
    mount, ok = m.mounts[id]
    m.RUnlock()
    return mount, ok
}

func main() {
    mounts := MountpointCollection{mounts: make(map[string]*Mountpoint)}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")

        switch r.Method {
            case http.MethodPost:
                if _, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                mount := mounts.NewMountpoint(r.URL.Path)
                log.Println("Mountpoint connected:", r.URL.Path)

                data := make([]byte, 1024)
                _, err := r.Body.Read(data)
                for ; err == nil; _, err = r.Body.Read(data) {
                    mount.Write(data)
                    data = make([]byte, 1024)
                }

                log.Println("Mountpoint disconnected:", r.URL.Path, err)

                mount.Lock()
                for _, client := range mount.Clients {
                    client.Cancel()
                }
                mount.Unlock()

                mounts.DeleteMountpoint(r.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    w.Header().Set("X-Content-Type-Options", "nosniff")
                    ctx, cancel := context.WithCancel(r.Context())
                    client := Client{requestId, w, r, cancel}
                    mount.AddClient(&client)
                    log.Println("Accepted Client on mountpoint", r.URL.Path)

                    <-ctx.Done()
                    mount.DeleteClient(requestId)
                    log.Println("Client disconnected")
                } else {
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2103", nil))
}
