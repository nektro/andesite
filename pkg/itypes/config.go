package itypes

import (
	oauth2 "github.com/nektro/go.oauth2"
)

type Config struct {
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

func (c *Config) GetDiscordClient() *oauth2.AppConf {
	for _, item := range c.Clients {
		if item.For == oauth2.ProviderIDMap["discord"].ID {
			return &item
		}
	}
	return &oauth2.AppConf{}
}
