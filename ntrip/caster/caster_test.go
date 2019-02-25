package caster_test

import (
    "testing"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "net/http/httptest"
    "net/http"
    "reflect"
    "bytes"
    "errors"
    "io"
    "time"
)

type MockAuth struct {}
func (ma MockAuth) Authorize(c *caster.Connection) error {
    if c.Request.URL.Path == "/401" { return errors.New("Unauthorized") }
    return nil
}

var (
    cast = caster.Caster{
        Mounts: make(map[string]*caster.Mountpoint),
        Authorizer: MockAuth{},
        Timeout: 1 * time.Second,
    }
    data = []byte("test data")
    conn = caster.NewConnection(nil, httptest.NewRequest("POST", "/TEST", bytes.NewReader(data)))
    mount = &caster.Mountpoint{
        Source: conn,
        Subscribers: make(map[string]caster.Subscriber),
    }
)

func TestRequestHandlerAuthorizedPOST(t *testing.T) {
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("POST", "/200", nil))
    if rr.Code != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusOK)
    }
}

func TestRequestHandlerAuthorizedGET(t *testing.T) {
    cast.AddMountpoint(mount)
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("GET", mount.Source.Request.URL.Path, nil))
    if rr.Code != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusOK)
    }
    cast.DeleteMountpoint(mount.Source.Request.URL.Path)
}

func TestRequestHandlerUnauthorized(t *testing.T) {
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("POST", "/401", nil))
    if rr.Code != http.StatusUnauthorized {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusUnauthorized)
    }

    rr = httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("GET", "/401", nil))
    if rr.Code != http.StatusUnauthorized {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusUnauthorized)
    }
}

func TestRequestHandlerStatusConflict(t *testing.T) {
    cast.AddMountpoint(mount)
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("POST", mount.Source.Request.URL.Path, nil))
    if rr.Code != http.StatusConflict {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusConflict)
    }
    cast.DeleteMountpoint(mount.Source.Request.URL.Path)
}

func TestRequestHandlerStatusNotFound(t *testing.T) {
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("GET", "/404", nil))
    if rr.Code != http.StatusNotFound {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusNotFound)
    }
}

func TestRequestHandlerStatusNotImplemented(t *testing.T) {
    rr := httptest.NewRecorder()
    cast.RequestHandler(rr, httptest.NewRequest("HEAD", "/501", nil))
    if rr.Code != http.StatusNotImplemented {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusNotImplemented)
    }
}

func TestCasterMountpointMethods(t *testing.T) {
    cast.AddMountpoint(mount)
    if m, exists := cast.Mounts[mount.Source.Request.URL.Path]; m != mount || !exists {
        t.Errorf("failed to add mountpoint")
    }
    if m := cast.GetMountpoint(mount.Source.Request.URL.Path); m != mount {
        t.Errorf("failed to get mountpoint")
    }
    cast.DeleteMountpoint(mount.Source.Request.URL.Path)
    if _, exists := cast.Mounts[mount.Source.Request.URL.Path]; exists {
        t.Errorf("failed to delete mountpoint")
    }
}

func TestMountpointMethods(t *testing.T) {
    err := mount.ReadSourceData()
    if err.Error() != "EOF" {
        t.Errorf("unexpected error while reading source data - " + err.Error())
    }

    client := caster.NewConnection(nil, nil)
    mount.RegisterSubscriber(client)

    err = mount.Broadcast(1 * time.Second)
    if err.Error() != "Timeout reading from source" {
        t.Errorf("unexpected error in Broadcast - " + err.Error())
    }

    select {
    case d := <-client.Channel():
        if !reflect.DeepEqual(d, data) {
            t.Errorf("read incorrect data from client channel: " + string(d))
        }
    default:
        t.Errorf("failed to read data from client channel")
    }

    mount.DeregisterSubscriber(client)
    if _, exists := mount.Subscribers[client.Id()]; exists {
        t.Errorf("failed to deregister subscriber")
    }
}

func TestHTTPServer(t *testing.T) {
    go cast.ListenHTTP(":2101")
    r, w := io.Pipe()
    http.Post("http://0.0.0.0:2101/test", "", r)
    go func() {
        for i := 0; i < 10; i += 1 {
            w.Write([]byte(time.Now().String() + "\r\n"))
            time.Sleep(100 * time.Millisecond)
        }
    }()

    resp, err := http.Get("http://0.0.0.0:2101/test")
    if err != nil {
        t.Errorf("failed to connect to mountpoint")
    }
    if resp.StatusCode != 200 {
        t.Errorf("handler returned wrong status code: got %v want %v",
            resp.StatusCode, http.StatusOK)
    }

    resp.Body.Read([]byte{})
    resp.Body.Close()
}
