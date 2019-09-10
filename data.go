package main

import (
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/mitchellh/go-homedir"
	oauth2 "github.com/nektro/go.oauth2"

	"github.com/nektro/andesite/internal/itypes"
)

const (
	DiscordAPI = "https://discordapp.com/api/v6"
)

var (
	config          *itypes.Config
	oauth2AppConfig *oauth2.AppConf
	oauth2Provider  itypes.Oauth2Provider
	randomKey       = securecookie.GenerateRandomKey(32)
	store           = sessions.NewCookieStore(randomKey)
	homedirPath, _  = homedir.Dir()
)
