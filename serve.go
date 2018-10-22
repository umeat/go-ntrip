package main

import (
    "fmt"
    "net/url"
    "github.com/benburkert/http"
    "io"
    "io/ioutil"
    "time"
)

func main() {
    ntripCaster, _ := url.Parse("http://127.0.0.1:2101/bar")

    read, write := io.Pipe()

    go func() {
        for {
            fmt.Fprintf(write, "%s\n\r", time.Now().UTC())
            time.Sleep(500 * time.Millisecond)
        }
    }()

    req := &http.Request{
        Method: "POST",
        ProtoMajor: 1,
        ProtoMinor: 1,
        URL: ntripCaster,
        TransferEncoding: []string{"chunked"},
        Body: read,
        Header: make(map[string][]string),
    }

    //req.Header.Set("Transfer-Encoding", "chunked")
    req.Header.Set("User-Agent", "NTRIP GoClient")
    req.Header.Set("Ntrip-Version", "Ntrip/2.0")

    httpClient := http.DefaultClient

    resp, err := httpClient.Do(req)
    if nil != err {
            fmt.Println("error =>", err.Error())
            return
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if nil != err {
        fmt.Println("error =>", err.Error())
    } else {
        fmt.Println(string(body))
    }
}
