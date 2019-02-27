package caster

import (
    "net/http"
    "github.com/satori/go.uuid"
)

// A client HTTP(S) request, implements Subscriber interface
type Connection struct {
    id string
    channel chan []byte
    Writer http.ResponseWriter
    Request *http.Request
}

func NewConnection(w http.ResponseWriter, r *http.Request) (conn *Connection) {
    requestId := uuid.Must(uuid.NewV4(), nil).String()
    return &Connection{requestId, make(chan []byte, 10), w, r}
}

func (conn *Connection) Id() string {
    return conn.id
}

func (conn *Connection) Channel() chan []byte {
    return conn.channel
}
