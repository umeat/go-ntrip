package main

import (
    log "github.com/sirupsen/logrus"
    "github.com/micro/go-config"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "github.com/umeat/go-ntrip/ntrip/caster/authorizers"
    "time"
)

var (
    ntripcaster = caster.Caster{
        Mounts: make(map[string]*caster.Mountpoint),
        Timeout: 5 * time.Second,
    } // TODO: Hide behind NewCaster which can include a DefaultAuthenticator
    conf Config
)

func main() {
    log.SetFormatter(&log.JSONFormatter{})

    config.LoadFile("cmd/ntripcaster/caster.json") // TODO: Take as command line parameter
    config.Scan(&conf)

    ntripcaster.Authorizer = authorizers.NewCognitoAuthorizer(conf.Cognito.UserPoolId, conf.Cognito.ClientId)

    go func() { panic(ntripcaster.ListenHTTPS(conf.Https.Port, conf.Https.CertificateFile, conf.Https.PrivateKeyFile)) }()
    panic(ntripcaster.ListenHTTP(conf.Http.Port))
}
