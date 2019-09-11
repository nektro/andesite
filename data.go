package main

import (
	"github.com/mitchellh/go-homedir"

	"github.com/nektro/andesite/internal/itypes"
)

const (
	RequiredConfigVersion = 2
	DiscordAPI            = "https://discordapp.com/api/v6"
)

var (
	config         *itypes.Config
	homedirPath, _ = homedir.Dir()
)
