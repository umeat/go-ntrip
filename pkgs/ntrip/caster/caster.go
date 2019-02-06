package caster

import (
    "net/http"
    log "github.com/sirupsen/logrus"
    "github.com/satori/go.uuid"
    "fmt"
)

func Serve(auth Authenticator) { // Still not sure best how to lay out this package - what belongs in cmd/ntripcaster vs what belongs here?
    log.SetFormatter(&log.JSONFormatter{})

    mounts := MountpointCollection{Mounts: make(map[string]*Mountpoint)}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        logger := log.WithFields(log.Fields{
            "request_id": requestId,
            "path": r.URL.Path,
            "method": r.Method,
            "source_ip": r.RemoteAddr,
        }) // Should this logger be an attribute of the Connection type?

        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")
        w.Header().Set("Content-Type", "application/octet-stream")

        // Not sure how large to make the buffered channel
        client := &Connection{requestId, make(chan []byte, 50), r, w}
        defer client.Request.Body.Close()

        if err := auth.Authenticate(client); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            logger.Error("Unauthorized")
            return
        }

        switch client.Request.Method {
            case http.MethodPost:
                mount, err := mounts.NewMountpoint(client) // Should probably construct the mountpoint first then pass it to mounts.AddMountpoint - will be relevant if we make an interface for mount sources (if we want a different kind of source)
                if err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                //client.Writer.(http.Flusher).Flush()

                logger.Info("Mountpoint Connected")
                serverChan := make(chan []byte, 5)
                go func(serverChan chan []byte) {
                    buf := make([]byte, 4096)
                    nbytes, err := mount.Source.Request.Body.Read(buf)
                    for ; err == nil; nbytes, err = mount.Source.Request.Body.Read(buf) {
                        serverChan <- buf[:nbytes]
                        buf = make([]byte, 4096)
                    }
                }(serverChan)

                var clients []chan []byte
                for {
                    select {
                    case c, _ := <-mount.Clients:
                        clients = append(clients, c)
                    case data, _ := <-serverChan:
                        for _, c := range clients {
                            select {
                            case c <- data:
                                continue
                            default:
                                continue
                            }
                        }
                    case <-mount.Source.Request.Context().Done():
                        logger.Info("Mountpoint Disconnected")
                        mounts.DeleteMountpoint(mount.Path)
                        return
                    }
                }

            case http.MethodGet:
                if mount, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    logger.Info("Accepted Client")
                    mount.Clients <- client.Channel
                    for {
                        select {
                        case data, _ := <-client.Channel:
                            fmt.Fprintf(client.Writer, "%s", data)
                            client.Writer.(http.Flusher).Flush()
                        case <-client.Request.Context().Done():
                            logger.Info("Client Disconnected - client closed connection")
                            return
                        case <-mount.Source.Request.Context().Done():
                            logger.Info("Client Disconnected - mountpoint closed")
                            return
                        }
                    }
                } else {
                    logger.Error("Not Found")
                    w.WriteHeader(http.StatusNotFound)
                }

            default:
                logger.Error("Not Implemented")
                w.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
