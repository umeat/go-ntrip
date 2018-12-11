package caster

import (
    "net/http"
    log "github.com/sirupsen/logrus"
    "context"
    "github.com/satori/go.uuid"
)

type Authenticator interface {
    Authenticate(*Connection) error
}

func Serve(auth Authenticator) { // Still not sure best how to lay out this package - what belongs in cmd/ntripcaster vs what belongs here?
    log.SetFormatter(&log.JSONFormatter{})

    mounts := MountpointCollection{Mounts: make(map[string]*Mountpoint)}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestId := uuid.Must(uuid.NewV4()).String()
        logger := log.WithFields(log.Fields{
            "requestId": requestId,
            "path": r.URL.Path,
            "method": r.Method,
        }) // Should this logger be an attribute of the Connection type?

        w.Header().Set("X-Request-Id", requestId)
        w.Header().Set("Ntrip-Version", "Ntrip/2.0")
        w.Header().Set("Server", "NTRIP GoCaster")
        w.Header().Set("Content-Type", "application/octet-stream")

        ctx, cancel := context.WithCancel(r.Context())
        // Not sure how large to make the buffered channel
        client := &Connection{requestId, make(chan []byte, 5), r, w, ctx, cancel}
        defer client.Request.Body.Close()

        if err := auth.Authenticate(client); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            logger.Error("Unauthorized")
            return
        }

        switch client.Request.Method {
            case http.MethodPost:
                <-client.Write([]byte("\r\n")) // Write behaves strangely if run asynchronously - however it may block forever in some cases

                mount, err := mounts.NewMountpoint(client) // Should probably construct the mountpoint first then pass it to mounts.AddMountpoint - will be relevant if we make an interface for mount sources (if we want a different kind of source)
                if err != nil {
                    w.WriteHeader(http.StatusConflict)
                    return
                }

                logger.Info("Mountpoint Connected")
                err = mount.Broadcast()

                logger.Info("Mountpoint Disconnected - ", err)
                mounts.DeleteMountpoint(mount.Source.Request.URL.Path)

            case http.MethodGet:
                if mount, exists := mounts.GetMountpoint(r.URL.Path); exists {
                    logger.Info("Accepted Client")
                    err := client.Subscribe(mount)
                    logger.Info("Client Disconnected - ", err)
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
