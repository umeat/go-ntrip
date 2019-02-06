package caster

import (
    "errors"
    "sync"
    "net/http"
)

type Authenticator interface {
    Authenticate(*Connection) error
}

type Connection struct {
    Id string
    Channel chan []byte
    Request *http.Request
    Writer http.ResponseWriter
}


type Mountpoint struct {
    sync.RWMutex
    Path string
    Source *Connection
    Clients chan chan []byte
}


type MountpointCollection struct {
    sync.RWMutex
    Mounts map[string]*Mountpoint
}

func (m MountpointCollection) NewMountpoint(source *Connection) (mount *Mountpoint, err error) {
    path := source.Request.URL.Path
    m.Lock()
    if _, ok := m.Mounts[path]; ok {
        m.Unlock()
        return mount, errors.New("Mountpoint in use")
    }

    mount = &Mountpoint{
        Path: path,
        Source: source,
        Clients: make(chan chan []byte, 4),
    }

    m.Mounts[path] = mount
    m.Unlock()
    return mount, nil
}

func (m MountpointCollection) DeleteMountpoint(id string) {
    m.Lock()
    delete(m.Mounts, id)
    m.Unlock()
}

func (m MountpointCollection) GetMountpoint(id string) (mount *Mountpoint, ok bool) {
    m.RLock()
    mount, ok = m.Mounts[id]
    m.RUnlock()
    return mount, ok
}
