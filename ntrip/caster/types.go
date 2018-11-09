package caster

import (
    "errors"
    "sync"
    "context"
    "net/http"
)

type Client struct {
    Id string
    Channel chan []byte
    Request *http.Request
    Writer http.ResponseWriter
    Context context.Context
    Cancel context.CancelFunc
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

func (mount *Mountpoint) Write(data []byte) {
    mount.RLock()
    for _, client := range mount.Clients {
        client.Channel <- data // Can this blow up?
    }
    mount.RUnlock()
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
