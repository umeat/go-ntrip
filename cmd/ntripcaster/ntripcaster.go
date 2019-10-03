package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/umeat/go-ntrip/ntrip/caster"
	"github.com/umeat/go-ntrip/ntrip/caster/authorizers"
	"github.com/spf13/viper"
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

	conf := Config{}
	viper.SetConfigFile(*configFile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		panic(err)
	}

	ntripcaster.Authorizer = authorizers.NewCognitoAuthorizer(conf.Cognito.UserPoolID, conf.Cognito.ClientID)

	panic(ntripcaster.ListenHTTP(conf.HTTP.Port))
}
