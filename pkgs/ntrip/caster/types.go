package caster

import (
    "errors"
    "sync"
    "context"
    "net/http"
    "fmt"
    "time"
)

type Connection struct {
    Id string
    Channel chan []byte
    Request *http.Request
    Writer http.ResponseWriter
    Context context.Context
    Cancel context.CancelFunc
}

func (conn *Connection) Listen() { // I think this a bit of a misnomer - sounds like we're waiting for the client to send us data
    conn.Writer.Header().Set("X-Content-Type-Options", "nosniff")

    for conn.Context.Err() != context.Canceled {
        select {
            case data := <-conn.Channel:
                select {
                    case <-conn.Write(data):
                        continue
                    case <-time.After(30 * time.Second):
                        return
                }

            case <-time.After(10 * time.Second):
                return
        }
    }
}

func (conn *Connection) Write(data []byte) chan bool {
    c := make(chan bool)
    go func() {
        fmt.Fprintf(conn.Writer, "%s", data)
        conn.Writer.(http.Flusher).Flush()
        c <- true
    }()
    return c
}


type Mountpoint struct {
    sync.RWMutex
    Path string
    Source *Connection
    Clients map[string]*Connection
}

func (mount *Mountpoint) AddClient(client *Connection) {
    mount.Lock()
    mount.Clients[client.Id] = client
    mount.Unlock()
}

func (mount *Mountpoint) DeleteClient(id string) {
    mount.Lock()
    delete(mount.Clients, id)
    mount.Unlock()
}

func (mount *Mountpoint) Write(data []byte) {
    mount.RLock()
    for _, client := range mount.Clients {
        select {
            case client.Channel <- data: // Can this blow up? Do I need to do :nbytes?
                continue
            default:
                continue
        }
    }
    mount.RUnlock()
}

func (mount *Mountpoint) Broadcast() { // needs a better name - should return the error
    buf := make([]byte, 1024)
    nbytes, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; nbytes, err = mount.Source.Request.Body.Read(buf) {
        go mount.Write(buf[:nbytes]) // Not convinced that this will do anything
        buf = make([]byte, 1024)
    }

    mount.Lock()
    for _, client := range mount.Clients {
        client.Cancel()
    }
    mount.Unlock()
}


type MountpointCollection struct {
    sync.RWMutex
    mounts map[string]*Mountpoint
}

func (m MountpointCollection) NewMountpoint(source *Connection) (mount *Mountpoint, err error) {
    path := source.Request.URL.Path
    m.Lock()
    if _, ok := m.mounts[path]; ok {
        m.Unlock()
        return mount, errors.New("Mountpoint in use")
    }

    mount = &Mountpoint{
        Path: path,
        Source: source,
        Clients: make(map[string]*Connection),
    }

    m.mounts[path] = mount
    m.Unlock()
    return mount, nil
}

func (m MountpointCollection) DeleteMountpoint(id string) {
    m.Lock()
    delete(m.mounts, id)
    m.Unlock()
}

func (m MountpointCollection) GetMountpoint(id string) (mount *Mountpoint, ok bool) {
    m.RLock()
    mount, ok = m.mounts[id]
    m.RUnlock()
    return mount, ok
}
