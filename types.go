package main

import (
	"net/http"

	"github.com/nektro/go.oauth2"
)

//
type OAuth2CallBackResponse struct {
	AccessToken string `json:"access_token"`
}

//
type UserAccessRow struct {
	id   int
	user int
	path string
}

//
type UserRow struct {
	id        int
	snowflake string
	admin     bool
	name      string
}

//
type ShareRow struct {
	id   int
	hash string
	path string
}

// Middleware provides a convenient mechanism for augmenting HTTP requests
// entering the application. It returns a new handler which may perform various
// operations and should finish by calling the next HTTP handler.
//
// @from https://gist.github.com/gbbr/dc731df098276f1a135b343bf5f2534a
type Middleware func(next http.HandlerFunc) http.HandlerFunc

//
type RootDirType string

//
const (
	RootTypeDir  RootDirType = "dir"
	RootTypeHttp             = "http"
)

type Config struct {
	Root      string            `json:"root"`
	Public    string            `json:"public"`
	Port      int               `json:"port"`
	Themes    []string          `json:"themes"`
	HTTPBase  string            `json:"base"`
	SearchOn  bool              `json:"search_on"`
	Auth      string            `json:"auth"`
	Discord   *ConfigIDP        `json:"discord"`
	Reddit    *ConfigIDP        `json:"reddit"`
	GitHub    *ConfigIDP        `json:"github"`
	Google    *ConfigIDP        `json:"google"`
	Facebook  *ConfigIDP        `json:"facebook"`
	Microsoft *ConfigIDP        `json:"microsoft"`
	Providers []oauth2.Provider `json:"providers"`
	CustomIds []ConfigIDP       `json:"custom"`
}

type ConfigIDP struct {
	Auth   string `json:"auth"`
	ID     string `json:"id"`
	Secret string `json:"secret"`
}
//

type Oauth2Provider struct {
	IDP oauth2.Provider
	DbP string
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
