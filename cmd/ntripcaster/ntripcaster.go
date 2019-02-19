package main

import (
    "github.com/umeat/go-ntrip/pkgs/ntrip/caster"
    "os"
)

func main() {
    authorizer, _ := caster.NewCognitoAuthorizer(os.Getenv("COGNITO_USER_POOL_ID"), os.Getenv("COGNITO_CLIENT_ID")) // TODO: take relevant variables from Config
    caster.Serve(authorizer) // TODO: Pass in Config object - maybe https://micro.mu/docs/go-config.html#config
}
