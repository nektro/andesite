package main

import (
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/nektro/go-util/logger"
	"github.com/nektro/go-util/sqlite"
	"github.com/nektro/go-util/types"
)

var (
	config          *Config
	oauth2AppConfig *ConfigIDP
	oauth2Provider  Oauth2Provider
	database        *sqlite.DB
	wwFFS           types.MultiplexFileSystem
	randomKey       = securecookie.GenerateRandomKey(32)
	store           = sessions.NewCookieStore(randomKey)
	log             = logger.New()
)
