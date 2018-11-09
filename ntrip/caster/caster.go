package caster

import (
    "fmt"
    "net/http"
    "log"
    "context"
    "github.com/satori/go.uuid"
)

func ServeMountpoint(mount *Mountpoint) { // Should this be a method of Mountpoint?
    fmt.Fprintf(mount.Source.Writer, "\r\n")
    mount.Source.Writer.(http.Flusher).Flush()
    log.Println("Mountpoint connected:", mount.Source.Request.URL.Path)

    buf := make([]byte, 1024)
    _, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; _, err = mount.Source.Request.Body.Read(buf) {
        mount.Write(buf)
        buf = make([]byte, 1024)
    }

    log.Println("Mountpoint disconnected:", mount.Source.Request.URL.Path, err)

    mount.Lock()
    for _, client := range mount.Clients {
        client.Cancel()
    }
    mount.Unlock()
}

func ServeClient(client *Client) {
    client.Writer.Header().Set("X-Content-Type-Options", "nosniff")
    log.Println("Accepted Client on mountpoint", client.Request.URL.Path)

    for Client.Context.Err() != context.Canceled {
        select {
            case data := <-client.Channel:
                fmt.Fprintf(client.Writer, "%s", data)
                client.Writer.(http.Flusher).Flush()
            default:
                break
        }
    }

    mount.DeleteClient(requestId)
    log.Println("Client disconnected", client.Id)
}

func Serve() {
    mounts := MountpointCollection{mounts: make(map[string]*Mountpoint)}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")

        ctx, cancel := context.WithCancel(r.Context())
        // Not sure how large to make the buffered channel
        client := Client{requestId, make(chan []byte, 5), r, w, ctx, cancel}

        switch r.Method {
            case http.MethodPost:
                mount, err := mounts.NewMountpoint(client)
                if err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                ServeMountpoint(mount)
                mounts.DeleteMountpoint(r.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    mount.AddClient(&client) // Can this fail?
                    ServeClient(&client)
                } else {
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
