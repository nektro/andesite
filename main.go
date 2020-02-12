package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/fsdb"
	"github.com/nektro/andesite/pkg/handler"
	"github.com/nektro/andesite/pkg/idata"

	"github.com/aymerick/raymond"
	"github.com/nektro/go-util/arrays/stringsu"
	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"
	"github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"

	_ "github.com/nektro/andesite/statik"
)

var (
	Version = "vMASTER"
)

func main() {
	idata.Version = etc.FixBareVersion(Version)
	util.Log("Initializing Andesite " + idata.Version + "...")
	etc.AppID = "andesite"

	pflag.IntVar(&idata.Config.Version, "version", idata.RequiredConfigVersion, "Config version to use.")
	pflag.StringVar(&idata.Config.Root, "root", "", "Path of root directory for files")
	pflag.IntVar(&idata.Config.Port, "port", 8000, "Port to open server on")
	pflag.StringVar(&idata.Config.HTTPBase, "base", "/", "Http Origin Path")
	pflag.StringVar(&idata.Config.Public, "public", "", "Public root of files to serve")
	pflag.StringArrayVar(&idata.Config.SearchOn, "enable-search", []string{}, "Set to a root ID to enable file search for that directory.")
	pflag.StringArrayVar(&idata.Config.SearchOff, "disable-search", []string{}, "Set to a root ID to disable file search for that directory.")
	flagDGS := pflag.String("discord-guild-id", "", "")
	flagDBT := pflag.String("discord-bot-token", "", "")
	etc.PreInit()

	etc.Init("andesite", &idata.Config, "./files/", db.SaveOAuth2InfoCb)

	//

	for i, item := range idata.Config.Clients {
		if item.For == "discord" {
			if len(*flagDGS) > 0 {
				idata.Config.Clients[i].Extra1 = *flagDGS
			}
			if len(*flagDBT) > 0 {
				idata.Config.Clients[i].Extra2 = *flagDBT
			}
		}
	}

	if idata.Config.Version == 0 {
		idata.Config.Version = 1
	}
	if idata.Config.Version != idata.RequiredConfigVersion {
		util.DieOnError(
			E(F("Current idata.Config.json version '%d' does not match required version '%d'.", idata.Config.Version, idata.RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", idata.RequiredConfigVersion),
		)
	}

	idata.Config.SearchOn = stringsu.Depupe(idata.Config.SearchOn)
	idata.Config.SearchOff = stringsu.Depupe(idata.Config.SearchOff)

	//
	// database initialization

	db.Init()

	db.Upgrade()

	//
	// graceful stop

	util.RunOnClose(func() {
		util.Log("Gracefully shutting down...")

		util.Log("Saving database to disk")
		db.DB.Close()

		util.Log("Done!")
	})

	//
	// handlebars helpers

	raymond.RegisterHelper("url_name", func(x string) string {
		return strings.Replace(url.PathEscape(x), "%2F", "/", -1)
	})
	raymond.RegisterHelper("add_i", func(a, b int) int {
		return a + b
	})

	//
	// http server setup

	http.HandleFunc("/test", handler.HandleTest)

	if len(idata.Config.Root) > 0 {
		idata.Config.Root, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(idata.Config.Root, "~", idata.HomedirPath)))
		util.Log("Sharing private files from " + idata.Config.Root)
		util.DieOnError(util.Assert(util.DoesDirectoryExist(idata.Config.Root), "Please pass a valid directory as a root parameter!"))
		idata.DataPathsPrv["files"] = idata.Config.Root

		http.HandleFunc("/files/", handler.HandleDirectoryListing(handler.HandleFileListing))
		http.HandleFunc("/regen_passkey", handler.HandleRegenPasskey)
		http.HandleFunc("/logout", handler.HandleLogout)
		http.HandleFunc("/open/", handler.HandleDirectoryListing(handler.HandleShareListing))

		http.HandleFunc("/admin", handler.HandleAdmin)
		http.HandleFunc("/admin/users", handler.HandleAdminUsers)

		http.HandleFunc("/api/access/create", handler.HandleAccessCreate)
		http.HandleFunc("/api/access/update", handler.HandleAccessUpdate)
		http.HandleFunc("/api/access/delete", handler.HandleAccessDelete)

		http.HandleFunc("/api/share/create", handler.HandleShareCreate)
		http.HandleFunc("/api/share/update", handler.HandleShareUpdate)
		http.HandleFunc("/api/share/delete", handler.HandleShareDelete)

		http.HandleFunc("/api/access_discord_role/create", handler.HandleDiscordRoleAccessCreate)
		http.HandleFunc("/api/access_discord_role/update", handler.HandleDiscordRoleAccessUpdate)
		http.HandleFunc("/api/access_discord_role/delete", handler.HandleDiscordRoleAccessDelete)
	}

	if len(idata.Config.Public) > 0 {
		idata.Config.Public, _ = filepath.Abs(filepath.Clean(strings.ReplaceAll(idata.Config.Public, "~", idata.HomedirPath)))
		util.Log("Sharing public files from", idata.Config.Public)
		util.DieOnError(util.Assert(util.DoesDirectoryExist(idata.Config.Public), "Public root directory does not exist. Aborting!"))
		idata.DataPathsPub["public"] = idata.Config.Public

		http.HandleFunc("/public/", handler.HandleDirectoryListing(handler.HandlePublicListing))
	}

	//
	// initialize file database in background

	http.HandleFunc("/search", handler.HandleSearch)
	http.HandleFunc("/api/search", handler.HandleSearchAPI)

	if len(idata.Config.SearchOn) > 0 {
		for _, item := range idata.Config.SearchOn {
			go fsdb.Init(idata.DataPathsPub, item)
			go fsdb.Init(idata.DataPathsPrv, item)
		}
	}
	if len(idata.Config.SearchOff) > 0 {
		for _, item := range idata.Config.SearchOff {
			fsdb.DeInit(idata.DataPathsPub, item)
			fsdb.DeInit(idata.DataPathsPrv, item)
		}
	}

	//
	// start http server

	etc.StartServer(idata.Config.Port)
}
