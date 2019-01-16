package ntrip

import (
    "io"
    "net/url"
    "github.com/benburkert/http"
)

func Server(ntripCasterUrl string, reader io.ReadCloser, username string, password string) (err error) {
    u, _ := url.Parse(ntripCasterUrl)
    req := &http.Request{
        Method: "POST",
        ProtoMajor: 1,
        ProtoMinor: 1,
        URL: u,
        TransferEncoding: []string{"chunked"},
        Body: reader,
        Header: make(map[string][]string),
    }

    req.Header.Set("User-Agent", "NTRIP GoClient")
    req.Header.Set("Ntrip-Version", "Ntrip/2.0")
    req.SetBasicAuth(username, password)

    go http.DefaultClient.Do(req)

    return err
}
