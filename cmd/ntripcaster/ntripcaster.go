package main

import (
    log "github.com/sirupsen/logrus"
    "github.com/micro/go-config"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "github.com/umeat/go-ntrip/ntrip/caster/authorizers"
)

var (
    ntripcaster = caster.Caster{Mounts: make(map[string]*caster.Mountpoint)} //TODO: Hide behind NewCaster which can include a DefaultAuthenticator
    conf Config
)

func main() {
    log.SetFormatter(&log.JSONFormatter{})

    config.LoadFile("cmd/ntripcaster/caster.json")
    config.Scan(&conf)

    ntripcaster.Authenticator, _ = authorizers.NewCognitoAuthorizer(conf.Cognito.UserPoolId, conf.Cognito.ClientId)

    go func() { panic(ntripcaster.ServeTLS(conf.Https.Port, conf.Https.CertificateFile, conf.Https.PrivateKeyFile)) }()
    panic(ntripcaster.Serve(conf.Http.Port))
}
