package main

import (
    "fmt"
    "net/http"
    "log"
    "bufio"
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
    Id string
    Mutex *sync.Mutex
    Clients map[string]Client
}

func NewMountpoint(id string) Mountpoint {
    return Mountpoint{id, &sync.Mutex{}, make(map[string]Client)}
}

func (mount *Mountpoint) AddClient(client Client) {
    mount.Mutex.Lock()
    mount.Clients[client.Id] = client
    mount.Mutex.Unlock()
}

func (mount *Mountpoint) DeleteClient(id string) {
    mount.Mutex.Lock()
    delete(mount.Clients, id)
    mount.Mutex.Unlock()
}

func (mount *Mountpoint) Write(data []byte) {
    mount.Mutex.Lock()
    for _, client := range mount.Clients {
        fmt.Fprintf(client.Writer, "%s", data)
        client.Writer.(http.Flusher).Flush()
    }
    mount.Mutex.Unlock()
}

func main() {
    mounts := make(map[string]*Mountpoint)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        w.Header().Set("X-Request-Id", requestId)

        switch r.Method {
            case http.MethodPost:
                if _, exists := mounts[r.URL.Path]; exists {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                mount := NewMountpoint(requestId)
                mounts[r.URL.Path] = &mount
                log.Println("Mountpoint connected:", r.URL.Path)

                reader := bufio.NewReader(r.Body)
                data, err := reader.ReadBytes('\n')
                for ; err == nil; data, err = reader.ReadBytes('\n') {
                    mount.Write(data)
                }

                log.Println("Mountpoint disconnected:", r.URL.Path, err)

                mount.Mutex.Lock()
                for _, client := range mount.Clients {
                    client.Cancel()
                }
                mount.Mutex.Unlock()

                delete(mounts, r.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts[r.URL.Path]; exists {
                    w.Header().Set("X-Content-Type-Options", "nosniff")
                    ctx, cancel := context.WithCancel(r.Context())
                    mount.AddClient(Client{requestId, w, r, cancel})
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
