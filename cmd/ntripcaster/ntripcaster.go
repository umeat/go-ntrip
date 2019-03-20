package main

import (
	"flag"
	"github.com/micro/go-config"
	log "github.com/sirupsen/logrus"
	"github.com/umeat/go-ntrip/ntrip/caster"
	"github.com/umeat/go-ntrip/ntrip/caster/authorizers"
	"time"
)

var (
	ntripcaster = caster.Caster{
		Mounts:  make(map[string]*caster.Mountpoint),
		Timeout: 5 * time.Second,
	} // TODO: Hide behind NewCaster which can include a DefaultAuthenticator
	conf Config
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	configFile := flag.String("config", "cmd/ntripcaster/caster.json", "Path to config file")
	flag.Parse()

	config.LoadFile(*configFile)
	config.Scan(&conf)

	ntripcaster.Authorizer = authorizers.NewCognitoAuthorizer(conf.Cognito.UserPoolID, conf.Cognito.ClientID)

	go func() { panic(ntripcaster.ListenHTTP(conf.HTTP.Port)) }()
	panic(ntripcaster.ListenHTTPS(conf.HTTPS.Port, conf.HTTPS.CertificateFile, conf.HTTPS.PrivateKeyFile))
}
