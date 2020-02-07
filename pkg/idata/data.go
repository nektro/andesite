package idata

import (
	"github.com/nektro/andesite/pkg/itypes"

	"github.com/mitchellh/go-homedir"
)

const (
	RequiredConfigVersion = 2
	DiscordAPI            = "https://discordapp.com/api/v6"
)

var (
	Version        = "vMASTER"
	Config         = new(itypes.Config)
	HomedirPath, _ = homedir.Dir()
	DataPaths      = map[string]string{}
)
