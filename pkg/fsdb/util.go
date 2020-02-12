package fsdb

import (
	"os"

	"github.com/nektro/go-util/util"
)

func hash(algo string, pathS string) string {
	f, _ := os.Open(pathS)
	defer f.Close()
	return util.HashStream(algo, f)
}
