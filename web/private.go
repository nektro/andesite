package web

import (
	"net/http"
	"path/filepath"
	"strings"

	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/web/handler"
)

func RegisterPrivate(mux *http.ServeMux) {
	config.Config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(config.Config.Root, "~", config.HomedirPath, -1)))
	Log("Sharing private files from " + config.Config.Root)
	DieOnError(Assert(DoesDirectoryExist(config.Config.Root), "Please pass a valid directory as a root parameter!"))
	config.DataPaths["files"] = config.Config.Root

	mux.HandleFunc("/admin", handler.HandleAdmin)
	mux.HandleFunc("/admin/users", handler.HandleAdminUsers)

	mux.HandleFunc("/api/access/delete", handler.HandleAccessDelete)
	mux.HandleFunc("/api/access/update", handler.HandleAccessUpdate)
	mux.HandleFunc("/api/access/create", handler.HandleAccessCreate)

	mux.HandleFunc("/api/share/create", handler.HandleShareCreate)
	mux.HandleFunc("/api/share/update", handler.HandleShareUpdate)
	mux.HandleFunc("/api/share/delete", handler.HandleShareDelete)

	mux.HandleFunc("/api/access_discord_role/create", handler.HandleDiscordRoleAccessCreate)
	mux.HandleFunc("/api/access_discord_role/update", handler.HandleDiscordRoleAccessUpdate)
	mux.HandleFunc("/api/access_discord_role/delete", handler.HandleDiscordRoleAccessDelete)

	mux.HandleFunc("/regen_passkey", handler.HandleRegenPasskey)
	mux.HandleFunc("/logout", handler.HandleLogout)

	mux.HandleFunc("/files/", handler.HandleDirectoryListing(handler.HandleFileListing))
	mux.HandleFunc("/open/", handler.HandleDirectoryListing(handler.HandleShareListing))
}
