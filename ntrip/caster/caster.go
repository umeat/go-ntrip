package caster

import (
    "net/http"
    log "github.com/sirupsen/logrus"
    "github.com/satori/go.uuid"
    "sync"
    "fmt"
    "errors"
)

// HTTP(S) server implementing the semantics of the NTRIPv2 protocol.
// A Caster could be described as a collection of Mountpoints.
// Sources POST (publish) streaming data to unique Mountpoints (URL Paths)
// on the Caster.
// Clients subscribe to streams via GET requests to Mountpoints.
type Caster struct {
    sync.RWMutex
    // A Collection of URL paths to which data is being streamed
    Mounts map[string]*Mountpoint
    // Caster calls Authorizer.Authorize for all HTTP(S) requests
    Authorizer Authorizer
}

// Starts HTTP server given a port in the format of the net/http library
func (caster Caster) ListenHTTP(port string) error {
    server := &http.Server{
        Addr: port,
        Handler: caster,
    }
    return server.ListenAndServe()
}

// Starts HTTPS server given a port in the format of the net/http library,
// a path to the certificate file, and a path to the private key file
func (caster Caster) ListenHTTPS(port, certificate, key string) error {
    server := &http.Server{
        Addr: port,
        Handler: caster,
    }
    return server.ListenAndServeTLS(certificate, key)
}

// Handler function for all incoming HTTP(S) requests.
// By allowing the Caster to implement the http.Handler interface, it can
// be used as the Handler for http.Server in the Caster Listen functions
func (caster Caster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    requestId := uuid.Must(uuid.NewV4(), nil).String()
    logger := log.WithFields(log.Fields{
        "request_id": requestId,
        "path": r.URL.Path,
        "method": r.Method,
        "source_ip": r.RemoteAddr,
    })

    conn := &Connection{requestId, make(chan []byte, 10), r, w}
    defer conn.Request.Body.Close()

    w.Header().Set("X-Request-Id", requestId)
    w.Header().Set("Ntrip-Version", "Ntrip/2.0")
    w.Header().Set("Server", "NTRIP GoCaster")
    w.Header().Set("Content-Type", "application/octet-stream")

    if err := caster.Authorizer.Authorize(conn); err != nil {
        w.WriteHeader(http.StatusUnauthorized)
        logger.Error("Unauthorized - ", err)
        return
    }

    switch conn.Request.Method {
    case http.MethodPost:
        w.Header().Set("Connection", "close") // only set Connection close for mountpoints
        mount := &Mountpoint{Source: conn, Subscribers: make(map[string]Subscriber)} // TODO: Hide behind NewMountpoint
        err := caster.AddMountpoint(mount)
        if err != nil {
            logger.Error("Mountpoint In Use")
            conn.Writer.WriteHeader(http.StatusConflict)
            return
        }

        conn.Writer.(http.Flusher).Flush()
        logger.Info("Mountpoint Connected")

        go mount.ReadSourceData()
        err = mount.Broadcast()

        logger.Info("Mountpoint Disconnected - " + err.Error())
        caster.DeleteMountpoint(mount.Source.Request.URL.Path)
        return

    case http.MethodGet:
        mount := caster.GetMountpoint(conn.Request.URL.Path)
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
                conn.Writer.(http.Flusher).Flush() // TODO: Add timeout on write
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

func (caster Caster) AddMountpoint(mount *Mountpoint) (err error) {
    caster.Lock()
    defer caster.Unlock()
    if _, ok := caster.Mounts[mount.Source.Request.URL.Path]; ok {
        return errors.New("Mountpoint in use")
    }

    caster.Mounts[mount.Source.Request.URL.Path] = mount
    return nil
}

func (caster Caster) DeleteMountpoint(id string) {
    caster.Lock()
    defer caster.Unlock()
    delete(caster.Mounts, id)
}

func (caster Caster) GetMountpoint(id string) (mount *Mountpoint) {
    caster.RLock()
    defer caster.RUnlock()
    return caster.Mounts[id]
}
