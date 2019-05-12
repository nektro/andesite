package main

import (
	"github.com/nektro/go.oauth2"
)

type Oauth2Provider struct {
	idp oauth2.Provider
	dbp string
}

var (
	Oauth2Providers = map[string]Oauth2Provider{
		"discord": Oauth2Provider{
			oauth2.ProviderDiscord,
			"",
		},
		"reddit": Oauth2Provider{
			oauth2.ProviderReddit,
			"1:",
		},
		"github": Oauth2Provider{
			oauth2.ProviderGitHub,
			"2:",
		},
		"google": Oauth2Provider{
			oauth2.ProviderGoogle,
			"3:",
		},
		"facebook": Oauth2Provider{
			oauth2.ProviderFacebook,
			"4:",
		},
		"microsoft": Oauth2Provider{
			oauth2.ProviderMicrosoft,
			"5:",
		},
	}
)
