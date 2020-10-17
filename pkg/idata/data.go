package idata

import (
	"strings"

	"github.com/nektro/andesite/pkg/itypes"

	"github.com/mitchellh/go-homedir"
	"github.com/nektro/go-util/arrays/stringsu"
	"github.com/nektro/go-util/types"
	sdrie "github.com/nektro/go.sdrie"
)

const (
	RequiredConfigVersion = 2
	DiscordAPI            = "https://discordapp.com/api/v6"
)

var (
	Config         = new(itypes.Config)
	HomedirPath, _ = homedir.Dir()
	DataPathsPub   = map[string]string{}
	DataPathsPrv   = map[string]string{}
	Hashes         = []string{"MD5", "SHA1", "SHA256", "SHA512", "SHA3_512", "BLAKE2b_512"}
	HashingSem     *types.Semaphore
	TempStore      = sdrie.New()
)

func DisableHash(alg string) {
	Hashes = stringsu.Filter(Hashes, func(s string) bool {
		return !strings.HasPrefix(strings.ToLower(s), alg)
	})
}

func RootForData(pathS string) string {
	for k, v := range DataPathsPub {
		if pathS == v {
			return k
		}
	}
	for k, v := range DataPathsPrv {
		if pathS == v {
			return k
		}
	}
	return ""
}
