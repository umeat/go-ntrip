package ntrip

import (
    "io"
    "net/url"
    "github.com/benburkert/http"
)

type Server struct {
    http.Request
}

func NewServer(ntripCasterUrl string, reader io.ReadCloser) (server *Server, err error) {
    u, err := url.Parse(ntripCasterUrl)
    server = &Server{
        Request: http.Request{
            URL: u,
            Body: reader,
            Method: "POST",
            ProtoMajor: 1,
            ProtoMinor: 1,
            TransferEncoding: []string{"chunked"},
            Header: make(map[string][]string),
        },
    }
    server.Header.Set("User-Agent", "NTRIP GoClient")
    server.Header.Set("Ntrip-Version", "Ntrip/2.0")
    return server, err
}

func (server *Server) Connect() (resp *http.Response, err error) {
    return http.DefaultClient.Do(&server.Request)
}
