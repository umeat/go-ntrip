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
        })

        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")
        w.Header().Set("Content-Type", "application/octet-stream")
        w.Header().Set("Connection", "close")

        client := &Connection{requestId, make(chan []byte, 10), r, w}
        defer client.Request.Body.Close()

        //if err := auth.Authenticate(client); err != nil {
        //    w.WriteHeader(http.StatusUnauthorized)
        //    logger.Error("Unauthorized")
        //    return
        //}

        switch client.Request.Method {
            case http.MethodPost:
                mount := &Mountpoint{
                    Source: client,
                    Clients: make(map[string]*Connection),
                }

                err := mounts.AddMountpoint(mount)
                if err != nil {
                    logger.Error("Mountpoint In Use")
                    client.Writer.WriteHeader(http.StatusConflict)
                    return
                }

                client.Writer.WriteHeader(http.StatusOK)
                logger.Info("Mountpoint Connected")

                go mount.ReadFromSource()
                mount.Broadcast()

                logger.Info("Mountpoint Disconnected")
                mounts.DeleteMountpoint(mount.Source.Request.URL.Path)

            case http.MethodGet:
                mount := mounts.GetMountpoint(client.Request.URL.Path)
                if mount == nil {
                    logger.Error("No Existing Mountpoint")
                    client.Writer.WriteHeader(http.StatusNotFound)
                    return
                }

                logger.Info("Accepted Client")
                mount.RegisterClient(client)
                for {
                    select {
                    case data, _ := <-client.Channel:
                        fmt.Fprintf(client.Writer, "%s", data)
                        client.Writer.(http.Flusher).Flush()
                    case <-client.Request.Context().Done():
                        mount.DeregisterClient(client)
                        logger.Info("Client Disconnected - client closed connection")
                        return
                    case <-mount.Source.Request.Context().Done():
                        logger.Info("Client Disconnected - mountpoint closed connection")
                        return
                    }
                }

            default:
                logger.Error("Request Method Not Implemented")
                client.Writer.WriteHeader(http.StatusNotImplemented)
        }
    })

    log.Fatal(http.ListenAndServe(":2101", nil))
}
