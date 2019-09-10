package main

import (
	"github.com/mitchellh/go-homedir"
	oauth2 "github.com/nektro/go.oauth2"

	"github.com/nektro/andesite/internal/itypes"
)

const (
	RequiredConfigVersion = 1
	DiscordAPI            = "https://discordapp.com/api/v6"
)

var (
	config          *itypes.Config
	oauth2AppConfig *oauth2.AppConf
	oauth2Provider  oauth2.Provider
	homedirPath, _  = homedir.Dir()
)
