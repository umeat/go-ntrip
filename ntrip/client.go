package ntrip

import (
    "net/url"
    "github.com/benburkert/http"
)

type Client struct {
    *http.Request
}

func NewClient(casterUrl string) (client *Client, err error) {
    u, err := url.Parse(casterUrl)
    client = &Client{
        Request: &http.Request{
            URL: u,
            Method: "GET",
            ProtoMajor: 1,
            ProtoMinor: 1,
            Header: make(map[string][]string),
        },
    }
    client.Header.Set("User-Agent", "NTRIP GoClient")
    client.Header.Set("Ntrip-Version", "Ntrip/2.0")
    return client, err
}

func (client *Client) Connect() (resp *http.Response, err error) {
    return http.DefaultClient.Do(client.Request)
}
