package web

import (
	"net/http"
	"path/filepath"

	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/fs"
	"github.com/nektro/andesite/web/handler"
)

func RegisterPublic(mux *http.ServeMux) {
	config.Config.Public, _ = filepath.Abs(config.Config.Public)
	Log("Sharing public files from", config.Config.Public)
	DieOnError(Assert(DoesDirectoryExist(config.Config.Public), "Public root directory does not exist. Aborting!"))
	config.DataPaths["public"] = config.Config.Public

	mux.HandleFunc("/public/", handler.HandleDirectoryListing(handler.HandlePublicListing, fs.LocalStorage))
}
