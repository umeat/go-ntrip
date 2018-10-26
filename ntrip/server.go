package ntrip

import (
    "io"
    "net/url"
    "github.com/benburkert/http"
)

func Server(ntripCasterUrl *url.URL, reader io.ReadCloser) (err error) {
    req := &http.Request{
        Method: "POST",
        ProtoMajor: 1,
        ProtoMinor: 1,
        URL: ntripCasterUrl,
        TransferEncoding: []string{"chunked"},
        Body: reader,
        Header: make(map[string][]string),
    }

    req.Header.Set("User-Agent", "NTRIP GoClient")
    req.Header.Set("Ntrip-Version", "Ntrip/2.0")

    go http.DefaultClient.Do(req)

    return err
}
