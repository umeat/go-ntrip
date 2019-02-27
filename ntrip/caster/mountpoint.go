package caster

import (
    "errors"
    "sync"
    "time"
)

// Represents an object which can subscribe to streams from a Mountpoint
type Subscriber interface {
    Id() string
    Channel() chan []byte
}

// POST requests to an endpoint result in the construction of a Mountpoint.
// Mountpoints can be subscribed to, Subscribers implement a Channel to which
// POSTed data is written.
type Mountpoint struct {
    sync.RWMutex
    // Connection from which data is received
    Source *Connection
    // A collection of Subscribers to send data to
    Subscribers map[string]Subscriber
}

// Read data from Source Request Body and write to Source.Channel
func (mount *Mountpoint) ReadSourceData() error {
    buf := make([]byte, 4096)
    nbytes, err := mount.Source.Request.Body.Read(buf)
    for ; err == nil; nbytes, err = mount.Source.Request.Body.Read(buf) {
        mount.Source.channel <- buf[:nbytes] // Can this block indefinitely
        buf = make([]byte, 4096)
    }
    return err
}

// Read data from Source.Channel and write to all registered Subscriber Channels
func (mount *Mountpoint) Broadcast(timeout time.Duration) error {
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

        case <-time.After(timeout):
            return errors.New("Timeout reading from source")

        case <-mount.Source.Request.Context().Done():
            return errors.New("Source closed connection")
        }
    }
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
