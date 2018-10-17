package main

import (
    "fmt"
    "net/http"
    "log"
    "bufio"
)

type Client struct {
    Writer http.ResponseWriter
    Request *http.Request
    Finished chan bool
}

func main() {
    clients := make(map[string][]Client)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
            case http.MethodPost:
                reader := bufio.NewReader(r.Body)
                data, err := reader.ReadBytes('\n')
                for ; err == nil; data, err = reader.ReadBytes('\n') {
                    for _, client := range clients[r.URL.Path] {
                        fmt.Fprintf(client.Writer, "%s", data)
                        client.Writer.(http.Flusher).Flush()
                    }
                }

                log.Println("Mountpoint disconnected:", r.URL.Path, err)
                for _, client := range clients[r.URL.Path] {
                    client.Finished <- true
                }

            case http.MethodGet:
                client := Client{w, r, make(chan bool)}
                w.Header().Set("X-Content-Type-Options", "nosniff")
                clients[r.URL.Path] = append(clients[r.URL.Path], client)
                log.Println("Accepted Client on mountpoint", r.URL.Path)

                <-client.Finished

            default:
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
