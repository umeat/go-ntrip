package ntrip

import (
    "net/url"
    "io"
    "github.com/benburkert/http"
)

func Client(ntripCasterUrl *url.URL, username string, password string) (reader io.ReadCloser, err error) {
    req := &http.Request{
        Method: "GET",
        ProtoMajor: 1,
        ProtoMinor: 1,
        URL: ntripCasterUrl,
        Header: make(map[string][]string),
    }

    req.Header.Set("User-Agent", "NTRIP GoClient")
    req.Header.Set("Ntrip-Version", "Ntrip/2.0")
    req.SetBasicAuth(username, password)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return reader, err
    }

    return resp.Body, nil
}
