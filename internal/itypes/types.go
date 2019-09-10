package itypes

import (
	"net/http"

	oauth2 "github.com/nektro/go.oauth2"
)

//
type UserRow struct {
	ID        int    `json:"id"`
	Snowflake string `json:"snowflake" sqlite:"text"`
	Admin     bool   `json:"admin" sqlite:"tinyint(1)"`
	Name      string `json:"name" sqlite:"text"`
	JoinedOn  string `json:"joined_on" sqlite:"text"`
	PassKey   string `json:"passkey" sqlite:"text"`
}

//
type UserAccessRow struct {
	ID   int    `json:"id"`
	User int    `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}

//
type ShareRow struct {
	ID   int    `json:"id"`
	Hash string `json:"hash" sqlite:"text"` // character(32)
	Path string `json:"path" sqlite:"text"`
}

//
type DiscordRoleAccessRow struct {
	ID        int    `json:"id"`
	GuildID   string `json:"guild_snowflake" sqlite:"text"`
	RoleID    string `json:"role_snowflake" sqlite:"text"`
	Path      string `json:"path" sqlite:"text"`
	GuildName string `json:"guild_name" sqlite:"text"`
	RoleName  string `json:"role_name" sqlite:"text"`
}

// Middleware provides a convenient mechanism for augmenting HTTP requests
// entering the application. It returns a new handler which may perform various
// operations and should finish by calling the next HTTP handler.
//
// @from https://gist.github.com/gbbr/dc731df098276f1a135b343bf5f2534a
type Middleware func(next http.HandlerFunc) http.HandlerFunc

type Config struct {
	Version   int               `json:"version"`
	Root      string            `json:"root"`
	Public    string            `json:"public"`
	Port      int               `json:"port"`
	Themes    []string          `json:"themes"`
	HTTPBase  string            `json:"base"`
	SearchOn  bool              `json:"search_on"`
	Auth      string            `json:"auth"`
	Discord   *oauth2.AppConf   `json:"discord"`
	Reddit    *oauth2.AppConf   `json:"reddit"`
	GitHub    *oauth2.AppConf   `json:"github"`
	Google    *oauth2.AppConf   `json:"google"`
	Facebook  *oauth2.AppConf   `json:"facebook"`
	Microsoft *oauth2.AppConf   `json:"microsoft"`
	Providers []oauth2.Provider `json:"providers"`
	CustomIds []oauth2.AppConf  `json:"custom"`
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
