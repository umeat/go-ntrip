package caster

import (
    "errors"
    "sync"
    "context"
    "net/http"
    "fmt"
)

type Client struct {
    Id string
    Channel chan []byte
    Request *http.Request
    Writer http.ResponseWriter
    Context context.Context
    Cancel context.CancelFunc
}

func (client *Client) Listen() {
    client.Writer.Header().Set("X-Content-Type-Options", "nosniff")

    for client.Context.Err() != context.Canceled {
        select {
            case data := <-client.Channel:
                fmt.Fprintf(client.Writer, "%s", data)
                client.Writer.(http.Flusher).Flush()
            default:
                break
        }
    }
}


type Mountpoint struct {
    sync.RWMutex
    Path string
    Source Client
    Clients map[string]*Client
}

func (mount *Mountpoint) AddClient(client *Client) {
    mount.Lock()
    mount.Clients[client.Id] = client
    mount.Unlock()
}

func (mount *Mountpoint) DeleteClient(id string) {
    mount.Lock()
    delete(mount.Clients, id)
    mount.Unlock()
}

func (mount *Mountpoint) Broadcast() { // needs a better name
    fmt.Fprintf(mount.Source.Writer, "\r\n")
    mount.Source.Writer.(http.Flusher).Flush()

    buf := make([]byte, 1024)
    _, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; _, err = mount.Source.Request.Body.Read(buf) {
        mount.RLock()
        for _, client := range mount.Clients {
            client.Channel <- buf // Can this blow up?
        }
        mount.RUnlock()
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

func (m MountpointCollection) NewMountpoint(source Client) (mount *Mountpoint, err error) {
    path := source.Request.URL.Path
    m.Lock()
    if _, ok := m.mounts[path]; ok {
        m.Unlock()
        return mount, errors.New("Mountpoint in use")
    }

    mount = &Mountpoint{
        Path: path,
        Source: source,
        Clients: make(map[string]*Client),
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
