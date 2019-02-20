package main

import (
    "github.com/micro/go-config"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "github.com/umeat/go-ntrip/ntrip/caster/authorizers"
)

type AdditionalConfig struct {
    Cognito CognitoConf
}

type CognitoConf struct {
    UserPoolId string
    ClientId string
}

var (
    ntripcaster = caster.Caster{Mounts: make(map[string]*caster.Mountpoint)} //TODO: Hide behind NewCaster which can include a DefaultAuthenticator
    conf AdditionalConfig
)

func main() {
    config.LoadFile("cmd/ntripcaster/caster.json")
    config.Scan(&ntripcaster.Config)

    // This is an example of how custom auth can receive config from the same source as Caster
    config.Scan(&conf)
    ntripcaster.Authenticator, _ = authorizers.NewCognitoAuthorizer(auth.Cognito.UserPoolId, auth.Cognito.ClientId)

    panic(ntripcaster.Serve())
}
