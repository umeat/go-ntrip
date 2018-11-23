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

func (conn *Connection) Subscribe(mount *Mountpoint) (err error) {
    mount.AddClient(conn)
    defer mount.DeleteClient(conn.Id)

    conn.Writer.Header().Set("X-Content-Type-Options", "nosniff")

    for conn.Context.Err() != context.Canceled {
        select {
            case data := <-conn.Channel:
                select {
                    case <-conn.Write(data):
                        continue
                    case <-time.After(30 * time.Second):
                        return errors.New("Timed out on write")
                }

            case <-time.After(30 * time.Second):
                return errors.New("Timed out reading from channel")
        }
    }

    return err
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

func (mount *Mountpoint) Publish(data []byte) {
    mount.RLock()
    for _, client := range mount.Clients {
        select {
            case client.Channel <- data:
                continue
            default:
                continue
        }
    }
    mount.RUnlock()
}

func (mount *Mountpoint) Broadcast() (err error) {
    buf := make([]byte, 1024)
    nbytes, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; nbytes, err = mount.Source.Request.Body.Read(buf) {
        go mount.Publish(buf[:nbytes])
        buf = make([]byte, 1024)
    }

    mount.Lock()
    for _, client := range mount.Clients {
        client.Cancel()
    }
    mount.Unlock()

    return err
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
