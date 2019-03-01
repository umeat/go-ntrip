package ntrip

import (
    "io"
    "net/url"
    "github.com/benburkert/http"
)

type Server struct {
    *http.Request
    writer *io.PipeWriter
}

func NewServer(ntripCasterUrl string) (server *Server, err error) {
    u, err := url.Parse(ntripCasterUrl)
    server = &Server{
        Request: &http.Request{
            URL: u,
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
    reader, writer := io.Pipe()
    server.Request.Body = reader
    server.writer = writer
    return http.DefaultClient.Do(server.Request)
}

func (server *Server) Write(data []byte) (n int, err error) {
    return server.writer.Write(data)
}
