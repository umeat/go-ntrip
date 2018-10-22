package main

import (
    "fmt"
    "net/http"
    "log"
    "bufio"
    "context"
    "github.com/satori/go.uuid"
)

type Client struct {
    Id string
    Writer http.ResponseWriter
    Request *http.Request
}

type Mountpoint struct {
    Id string
    Clients map[string]Client
}

func main() {
    mounts := make(map[string]Mountpoint)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()

        switch r.Method {
            case http.MethodPost:
                mounts[r.URL.Path] = Mountpoint{requestId, make(map[string]Client)}
                log.Println("Mountpoint connected:", r.URL.Path)

                reader := bufio.NewReader(r.Body)
                data, err := reader.ReadBytes('\n')
                for ; err == nil; data, err = reader.ReadBytes('\n') {
                    for _, client := range mounts[r.URL.Path].Clients {
                        fmt.Fprintf(client.Writer, "%s", data)
                        client.Writer.(http.Flusher).Flush()
                    }
                }

                log.Println("Mountpoint disconnected:", r.URL.Path, err)

                for _, client := range mounts[r.URL.Path].Clients {
                    _, cancel := context.WithCancel(client.Request.Context())
                    cancel()
                }

                delete(mounts, r.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts[r.URL.Path]; exists {
                    w.Header().Set("X-Content-Type-Options", "nosniff")
                    mount.Clients[requestId] = Client{requestId, w, r}
                    log.Println("Accepted Client on mountpoint", r.URL.Path)

                    <-r.Context().Done()
                    delete(mount.Clients, requestId)

                    log.Println("Client disconnected")
                } else {
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
