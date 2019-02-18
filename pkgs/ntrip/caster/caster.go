package caster

import (
    "net/http"
    log "github.com/sirupsen/logrus"
    "github.com/satori/go.uuid"
    "fmt"
)

var (
    mounts = MountpointCollection{Mounts: make(map[string]*Mountpoint)}
    // sourcetable = Sourcetable{....}
)

func Serve(auth Authenticator) { // TODO: Serve should take a Config object of some description
    log.SetFormatter(&log.JSONFormatter{})
    http.HandleFunc("/", RequestHandler)
    log.Fatal(http.ListenAndServe(":2101", nil))
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
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

    conn := &Connection{requestId, make(chan []byte, 10), r, w}
    defer conn.Request.Body.Close()

    //if err := auth.Authenticate(conn); err != nil {
    //    w.WriteHeader(http.StatusUnauthorized)
    //    logger.Error("Unauthorized")
    //    return
    //}

    switch conn.Request.Method {
        case http.MethodPost:
            w.Header().Set("Connection", "close") // only set Connection close for mountpoints
            mount := &Mountpoint{Source: conn, Subscribers: make(map[string]Subscriber)} // TODO: Hide behind NewMountpoint
            err := mounts.AddMountpoint(mount)
            if err != nil {
                logger.Error("Mountpoint In Use")
                conn.Writer.WriteHeader(http.StatusConflict)
                return
            }

            conn.Writer.(http.Flusher).Flush()
            logger.Info("Mountpoint Connected")

            go mount.ReadSourceData()
            mount.Broadcast()

            logger.Info("Mountpoint Disconnected")
            mounts.DeleteMountpoint(mount.Source.Request.URL.Path)
            return

        case http.MethodGet:
            mount := mounts.GetMountpoint(conn.Request.URL.Path)
            if mount == nil {
                logger.Error("No Existing Mountpoint") // Should probably reserve logger.Error for server errors
                conn.Writer.WriteHeader(http.StatusNotFound)
                return
            }

            logger.Info("Accepted Client Connection")
            mount.RegisterSubscriber(conn)
            for { // TODO: Come up with a Connection struct method name which makes sense for this
                select {
                case data, _ := <-conn.channel:
                    fmt.Fprintf(conn.Writer, "%s", data)
                    conn.Writer.(http.Flusher).Flush()
                case <-conn.Request.Context().Done():
                    mount.DeregisterSubscriber(conn)
                    logger.Info("Client Disconnected - client closed connection")
                    return
                case <-mount.Source.Request.Context().Done():
                    logger.Info("Client Disconnected - mountpoint closed connection")
                    return
                }
            }

        default:
            logger.Error("Request Method Not Implemented")
            conn.Writer.WriteHeader(http.StatusNotImplemented)
    }
}
