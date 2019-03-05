# go-ntrip
NTRIP Caster / Client / Server implementation in Go

[![Coverage Status](https://coveralls.io/repos/github/umeat/go-ntrip/badge.svg?branch=master)](https://coveralls.io/github/umeat/go-ntrip?branch=master)

### Installation
```
git clone https://github.com/umeat/go-ntrip && cd go-ntrip
go build ./...
go install ./...

# or
go get github.com/umeat/go-ntrip/...
```

### Run a Caster 
Application in `cmd/ntripcaster/` configurable with `cmd/ntripcaster/caster.conf`.

```
# Generate self signed certs for testing
openssl genrsa -out key.pem 2048
openssl req -new -x509 -sha256 -key server.key -out cert.pem -days 3650

ntripcaster &
curl https://localhost:2102/mount -d "TEST" -i -k -u username:password &
curl http://localhost:2101/mount -i
```
