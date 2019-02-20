package main

import (
    "github.com/micro/go-config"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "github.com/umeat/go-ntrip/ntrip/caster/authorizers"
)

type AuthConf struct {
    Cognito CognitoConf
}

type CognitoConf struct {
    UserPoolId string
    ClientId string
}

var (
    ntripcaster = caster.Caster{Mounts: make(map[string]*caster.Mountpoint)} //TODO: Hide behind NewCaster
    auth AuthConf
)

func main() {
    config.LoadFile("cmd/ntripcaster/caster.json")
    config.Scan(&ntripcaster.Config)

    // This is an example of how custom auth can receive config from the same source as Caster
    config.Scan(&auth)
    ntripcaster.Authenticator, _ = authorizers.NewCognitoAuthorizer(auth.Cognito.UserPoolId, auth.Cognito.ClientId)

    ntripcaster.Serve()
}
