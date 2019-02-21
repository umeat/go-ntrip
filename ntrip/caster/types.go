package caster

import (
    "errors"
    "sync"
    "net/http"
    "time"
)

type Subscriber interface {
    Id() string
    Channel() chan []byte
}


type Connection struct {
    id string
    channel chan []byte
    Request *http.Request
    Writer http.ResponseWriter
}

func (conn *Connection) Id() string {
    return conn.id
}

func (conn *Connection) Channel() chan []byte {
    return conn.channel
}


type Mountpoint struct {
    sync.RWMutex
    Source *Connection
    Subscribers map[string]Subscriber
}

func (mount *Mountpoint) RegisterSubscriber(subscriber Subscriber) {
    mount.Lock()
    defer mount.Unlock()
    mount.Subscribers[subscriber.Id()] = subscriber
}

func (mount *Mountpoint) DeregisterSubscriber(subscriber Subscriber) {
    mount.Lock()
    defer mount.Unlock()
    delete(mount.Subscribers, subscriber.Id())
}

func (mount *Mountpoint) ReadSourceData() { // Read data from Request Body and write to Source.Channel
    buf := make([]byte, 4096)
    nbytes, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; nbytes, err = mount.Source.Request.Body.Read(buf) {
        mount.Source.channel <- buf[:nbytes] // Can this block indefinitely
        buf = make([]byte, 4096)
    }
}

//TODO: Return error
func (mount *Mountpoint) Broadcast() { // Read data from Source.Channel and write to registered subscriber channels
    for {
        select {
        case data, _ := <-mount.Source.channel:
            mount.RLock()
            for _, subscriber := range mount.Subscribers {
                select {
                case subscriber.Channel() <- data:
                    continue
                default:
                    continue // The default case should not occur now that subscriber can be deregistered
                }
            }
            mount.RUnlock()

        case <-time.After(time.Second * 5):
            return

        case <-mount.Source.Request.Context().Done():
            return
        }
    }
}


//type MountpointCollection struct {
//    sync.RWMutex
//    Mounts map[string]*Mountpoint
//}

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


type Authorizer interface {
    Authorize(*Connection) error
}
