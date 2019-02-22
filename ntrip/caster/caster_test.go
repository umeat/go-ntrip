package caster_test

import (
    "testing"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "net/http/httptest"
)

type MockAuth struct {}
func (ma MockAuth) Authorize(c *caster.Connection) error { return nil }

var (
    cast = caster.Caster{
        Mounts: make(map[string]*caster.Mountpoint),
        Authorizer: MockAuth{},
    }
    conn = caster.NewConnection(nil, httptest.NewRequest("POST", "/TEST", nil))
    mount = &caster.Mountpoint{
        Source: conn,
        Subscribers: make(map[string]caster.Subscriber),
    }
)

func TestAddMountpoint(t *testing.T) {
    cast.AddMountpoint(mount)
    if cast.GetMountpoint(mount.Source.Request.URL.Path) != mount {
        t.Errorf("GetMountpoint returned different object")
    }
}
