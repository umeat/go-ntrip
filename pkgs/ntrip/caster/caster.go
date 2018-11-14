package caster

import (
    "net/http"
    "log"
    "context"
    "github.com/satori/go.uuid"
    "time"
)

func Serve() {
    mounts := MountpointCollection{mounts: make(map[string]*Mountpoint)}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")
        w.Header().Set("Content-Type", "application/octet-stream")

        ctx, cancel := context.WithCancel(r.Context())
        // Not sure how large to make the buffered channel
        client := &Connection{requestId, make(chan []byte, 5), r, w, ctx, cancel}

        switch r.Method {
            case http.MethodPost:
                // A POST client may not read any response from the server, in which case a flush may block
                select {
                    case <-client.Write([]byte("\r\n")):
                        break
                    case <-time.After(2 * time.Second):
                        break
                }

                mount, err := mounts.NewMountpoint(client)
                if err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                log.Println("Mountpoint connected:", mount.Source.Request.URL.Path)
                mount.Broadcast()

                log.Println("Mountpoint disconnected:", mount.Source.Request.URL.Path, err)
                mounts.DeleteMountpoint(mount.Source.Request.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    mount.AddClient(client)

                    log.Println("Accepted Client on mountpoint", client.Request.URL.Path, client.Id)
                    client.Listen()

                    log.Println("Client disconnected", client.Id)
                    mount.DeleteClient(client.Id)
                } else {
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
