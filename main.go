package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/fs"
	"github.com/nektro/andesite/handler"
	"github.com/nektro/andesite/search"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
	"github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)

func main() {
	Log("Initializing Andesite " + config.Version + "...")

	pflag.IntVar(&config.Config.Version, "version", config.RequiredConfigVersion, "Config version to use.")
	pflag.StringVar(&config.Config.Root, "root", "", "Path of root directory for files")
	pflag.IntVar(&config.Config.Port, "port", 8000, "Port to open server on")
	pflag.StringVar(&config.Config.HTTPBase, "base", "/", "Http Origin Path")
	pflag.StringVar(&config.Config.Public, "public", "", "Public root of files to serve")
	pflag.BoolVar(&config.Config.SearchOn, "enable-search", false, "Set to true to enable search database")
	flagDGS := pflag.String("discord-guild-id", "", "")
	flagDBT := pflag.String("discord-bot-token", "", "")
	etc.PreInit("andesite")

	etc.Init("andesite", &config.Config, "./files/", helperOA2SaveInfo)

	//

	for i, item := range config.Config.Clients {
		if item.For == "discord" {
			if len(*flagDGS) > 0 {
				config.Config.Clients[i].Extra1 = *flagDGS
			}
			if len(*flagDBT) > 0 {
				config.Config.Clients[i].Extra2 = *flagDBT
			}
		}
	}

	//

	if config.Config.Version == 0 {
		config.Config.Version = 1
	}
	if config.Config.Version != config.RequiredConfigVersion {
		DieOnError(
			E(F("Current config.Config.json version '%d' does not match required version '%d'.", config.Config.Version, config.RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", config.RequiredConfigVersion),
		)
	}

	//
	// database initialization

	etc.Database.CreateTableStruct("users", db.UserRow{})
	etc.Database.CreateTableStruct("access", db.UserAccessRow{})
	etc.Database.CreateTableStruct("shares", db.ShareRow{})
	etc.Database.CreateTableStruct("shares_discord_role", db.DiscordRoleAccessRow{})

	//
	// database upgrade (removing db prefixes in favor of provider column)

	prefixes := map[string]string{
		"reddit":    "1:",
		"github":    "2:",
		"google":    "3:",
		"facebook":  "4:",
		"microsoft": "5:",
	}
	for _, item := range db.QueryAllUsers() {
		for k, v := range prefixes {
			if strings.HasPrefix(item.Snowflake, v) {
				sn := item.Snowflake[len(v):]
				Log("[db-upgrade]", item.Snowflake, "is now", sn, "as", k)
				etc.Database.Build().Up("users", "snowflake", sn).Wh("id", item.IDS).Exe()
				etc.Database.Build().Up("users", "provider", k).Wh("id", item.IDS).Exe()
			}
		}
	}

	//
	// graceful stop

	RunOnClose(func() {
		Log("Gracefully shutting down...")

		Log("Saving database to disk")
		etc.Database.Close()

		if config.Config.SearchOn {
			Log("Closing filesystem watcher")
			search.Close()
		}

		Log("Done!")
	})

	//
	// initialize filesystem watching

	if config.Config.SearchOn {
		go search.InitFsWatcher()
	}

	//
	// handlebars helpers

	raymond.RegisterHelper("url_name", func(x string) string {
		return strings.Replace(url.PathEscape(x), "%2F", "/", -1)
	})
	raymond.RegisterHelper("add_i", func(a, b int) int {
		return a + b
	})

	//
	// http server setup and launch

	http.HandleFunc("/test", handler.HandleTest)

	if len(config.Config.Root) > 0 {
		config.Config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(config.Config.Root, "~", config.HomedirPath, -1)))
		Log("Sharing private files from " + config.Config.Root)
		DieOnError(Assert(DoesDirectoryExist(config.Config.Root), "Please pass a valid directory as a root parameter!"))
		config.DataPaths["files"] = config.Config.Root

		http.HandleFunc("/admin", handler.HandleAdmin)
		http.HandleFunc("/admin/users", handler.HandleAdminUsers)

		http.HandleFunc("/api/access/delete", handler.HandleAccessDelete)
		http.HandleFunc("/api/access/update", handler.HandleAccessUpdate)
		http.HandleFunc("/api/access/create", handler.HandleAccessCreate)

		http.HandleFunc("/api/share/create", handler.HandleShareCreate)
		http.HandleFunc("/api/share/update", handler.HandleShareUpdate)
		http.HandleFunc("/api/share/delete", handler.HandleShareDelete)

		http.HandleFunc("/api/access_discord_role/create", handler.HandleDiscordRoleAccessCreate)
		http.HandleFunc("/api/access_discord_role/update", handler.HandleDiscordRoleAccessUpdate)
		http.HandleFunc("/api/access_discord_role/delete", handler.HandleDiscordRoleAccessDelete)

		http.HandleFunc("/regen_passkey", handler.HandleRegenPasskey)
		http.HandleFunc("/logout", handler.HandleLogout)

		http.HandleFunc("/files/", handler.HandleDirectoryListing(handler.HandleFileListing, fs.LocalStorage))
		http.HandleFunc("/open/", handler.HandleDirectoryListing(handler.HandleShareListing, fs.LocalStorage))
	}

	if len(config.Config.Public) > 0 {
		config.Config.Public, _ = filepath.Abs(config.Config.Public)
		Log("Sharing public files from", config.Config.Public)
		DieOnError(Assert(DoesDirectoryExist(config.Config.Public), "Public root directory does not exist. Aborting!"))
		config.DataPaths["public"] = config.Config.Public

		http.HandleFunc("/public/", handler.HandleDirectoryListing(handler.HandlePublicListing, fs.LocalStorage))
	}

	handlerWrapper := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Server", "nektro/andesite")
		})
	}

	DieOnError(Assert(IsPortAvailable(config.Config.Port), F("Binding to port %d failed.", config.Config.Port)), "It may be taken or you may not have permission to. Aborting!")
	p := strconv.Itoa(config.Config.Port)
	Log("Initialization complete. Starting server on port " + p)

	http.ListenAndServe(":"+p, handlerWrapper(http.DefaultServeMux))
}

func helperOA2SaveInfo(w http.ResponseWriter, r *http.Request, provider string, id string, name string, resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["provider"] = provider
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Values[provider+"_access_token"] = resp["access_token"]
	sess.Values[provider+"_expires_in"] = resp["expires_in"]
	sess.Values[provider+"_refresh_token"] = resp["refresh_token"]
	sess.Save(r, w)
	db.QueryAssertUserName(provider, id, name)
	Log("[user-login]", provider, id, name)
}
