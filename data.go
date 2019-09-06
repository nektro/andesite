package main

import (
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/nektro/go-util/logger"

	"github.com/nektro/andesite/internal/itypes"
)

var (
	config          *itypes.Config
	oauth2AppConfig *itypes.ConfigIDP
	oauth2Provider  itypes.Oauth2Provider
	randomKey       = securecookie.GenerateRandomKey(32)
	store           = sessions.NewCookieStore(randomKey)
	log             = logger.New()
)
