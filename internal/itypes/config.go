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

func (c *Config) GetDiscordClient() *oauth2.AppConf {
	return c.Discord
}
