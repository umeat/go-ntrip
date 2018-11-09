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
    Channel chan []byte
    Cancel context.CancelFunc
}

type Mountpoint struct {
    sync.RWMutex
    Path string
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
    mount.RLock()
    for _, client := range mount.Clients {
        client.Channel <- data // Can this blow up?
    }
    mount.RUnlock()
}

type MountpointCollection struct {
    sync.RWMutex
    mounts map[string]*Mountpoint
}

func (m MountpointCollection) NewMountpoint(path string) (mount *Mountpoint, err error) {
    m.Lock()
    if _, ok := m.mounts[path] {
        return mount, errors.New("Mountpoint in use")
    }
    mount = &Mountpoint{Path: path, Clients: make(map[string]*Client)}
    m.mounts[path] = mount
    m.Unlock()
    return mount, nil
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
                mount, err := mounts.NewMountpoint(r.URL.Path)
                if err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                fmt.Fprintf(w, "\r\n")
                w.(http.Flusher).Flush()
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

                    // Not sure how large to make the buffered channel
                    client := Client{requestId, make(chan []byte, 5), cancel}
                    mount.AddClient(&client)
                    log.Println("Accepted Client on mountpoint", r.URL.Path)

                    for ctx.Err() != context.Canceled {
                        data := <-client.Channel
                        fmt.Fprintf(w, "%s", data)
                        w.(http.Flusher).Flush()
                    }

                    mount.DeleteClient(requestId)
                    log.Println("Client disconnected", client.Id)
                } else {
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
