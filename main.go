package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
	"github.com/spf13/pflag"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/search"
	"github.com/nektro/andesite/web"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)

func init() {
	pflag.IntVar(&config.Config.Version, "version", config.RequiredConfigVersion, "Config version to use.")
	pflag.StringVar(&config.Config.Root, "root", "", "Path of root directory for files")
	pflag.IntVar(&config.Config.Port, "port", 8000, "Port to open server on")
	pflag.StringVar(&config.Config.HTTPBase, "base", "/", "Http Origin Path")
	pflag.StringVar(&config.Config.Public, "public", "", "Public root of files to serve")
	pflag.BoolVar(&config.Config.SearchOn, "enable-search", false, "Set to true to enable search database")
}

func main() {
	Log("Initializing Andesite " + config.Version + "...")
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

	// database initialization
	db.Init()

	// database upgrade (removing db prefixes in favor of provider column)
	db.Upgrade()

	// graceful stop
	RunOnClose(Shutdown)

	// initialize filesystem watching
	if config.Config.SearchOn {
		go search.InitFsWatcher()
	}

	// handlebars helpers
	raymond.RegisterHelper("url_name", func(x string) string {
		return strings.Replace(url.PathEscape(x), "%2F", "/", -1)
	})
	raymond.RegisterHelper("add_i", func(a, b int) int {
		return a + b
	})

	// http server setup and launch
	muxer := web.NewMuxer()

	if len(config.Config.Root) > 0 {
		web.RegisterPrivate(muxer)
	}

	if len(config.Config.Public) > 0 {
		web.RegisterPublic(muxer)
	}

	handlerWrapper := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Server", "nektro/andesite")
		})
	}

	DieOnError(Assert(IsPortAvailable(config.Config.Port), F("Binding to port %d failed.", config.Config.Port)), "It may be taken or you may not have permission to. Aborting!")
	p := strconv.Itoa(config.Config.Port)
	Log("Initialization complete. Starting server on port " + p)

	http.ListenAndServe(":"+p, handlerWrapper(muxer))
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

func Shutdown() {
	Log("Gracefully shutting down...")

	Log("Saving database to disk")
	etc.Database.Close()

	if config.Config.SearchOn {
		Log("Closing filesystem watcher")
		search.Close()
	}

	Log("Done!")
}
