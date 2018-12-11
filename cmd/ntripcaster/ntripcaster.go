package main

import (
    "github.com/umeat/go-ntrip/pkgs/ntrip/caster"
)

func main() {
    authorizer, _ := caster.NewCognitoAuthorizer() // TODO: take relevant variables from Config
    caster.Serve(authorizer) // TODO: Pass in Config object - maybe https://micro.mu/docs/go-config.html#config
}
