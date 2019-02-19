package main

import (
    "github.com/micro/go-config"
    "github.com/umeat/go-ntrip/pkgs/ntrip"
    "github.com/umeat/go-ntrip/pkgs/ntrip/authorizers"
)

type AuthConf struct {
    Cognito CognitoConf
}

type CognitoConf struct {
    UserPoolId string
    ClientId string
}

var (
    caster = ntrip.Caster{Mounts: make(map[string]*ntrip.Mountpoint)} //TODO: Hide behind NewCaster
    auth AuthConf
)

func main() {
    config.LoadFile("cmd/ntripcaster/caster.json")
    config.Scan(&caster.Config)

    // This is an example of how custom auth can receive config from the same source as Caster
    config.Scan(&auth)
    caster.Authenticator, _ = authorizers.NewCognitoAuthorizer(auth.Cognito.UserPoolId, auth.Cognito.ClientId)

    caster.ServeTLS()
}
