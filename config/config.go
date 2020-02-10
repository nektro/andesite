package config

import (
	"github.com/mitchellh/go-homedir"
	oauth2 "github.com/nektro/go.oauth2"
)

const (
	RequiredConfigVersion = 2
	DiscordAPI            = "https://discordapp.com/api/v6"
)

var (
	Version        = "vMASTER"
	Config         = config{}
	HomedirPath, _ = homedir.Dir()
	DataPaths      = map[string]string{}
)

type config struct {
	Version   int               `json:"version"`
	Root      string            `json:"root"`
	Public    string            `json:"public"`
	Port      int               `json:"port"`
	Themes    []string          `json:"themes"`
	HTTPBase  string            `json:"base"`
	SearchOn  bool              `json:"search_on"`
	Clients   []oauth2.AppConf  `json:"clients"`
	Providers []oauth2.Provider `json:"providers"`
}

func (c *config) GetDiscordClient() *oauth2.AppConf {
	for _, item := range c.Clients {
		if item.For == "discord" {
			return &item
		}
	}
	return &oauth2.AppConf{}
}
