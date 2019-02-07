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
    Source *Connection
    Clients chan *Connection
}


type MountpointCollection struct {
    sync.RWMutex
    Mounts map[string]*Mountpoint
}

func (mc MountpointCollection) AddMountpoint(mount *Mountpoint) (err error) {
    mc.Lock()
    defer mc.Unlock()
    if _, ok := mc.Mounts[mount.Source.Request.URL.Path]; ok {
        return errors.New("Mountpoint in use")
    }

    mc.Mounts[mount.Source.Request.URL.Path] = mount
    return nil
}

func (m MountpointCollection) DeleteMountpoint(id string) {
    m.Lock()
    delete(m.Mounts, id)
    m.Unlock()
}

func (m MountpointCollection) GetMountpoint(id string) (mount *Mountpoint) {
    m.RLock()
    defer m.RUnlock()
    return m.Mounts[id]
}
