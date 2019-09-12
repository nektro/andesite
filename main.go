package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/rakyll/statik/fs"
	flag "github.com/spf13/pflag"

	"github.com/nektro/andesite/internal/idata"
	"github.com/nektro/andesite/internal/itypes"
	"github.com/nektro/andesite/internal/iutil"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)


func main() {
	Log("Initializing Andesite...")

	flagRoot := flag.String("root", "", "Path of root directory for files")
	flagPort := flag.Int("port", 0, "Port to open server on")
	flagBase := flag.String("base", "", "Http Origin Path")
	flagPublic := flag.String("public", "", "Public root of files to serve")
	flagSearch := flag.Bool("enable-search", false, "Set to true to enable search database")
	flag.Parse()

	//
	// parse options and find config

	etc.Init("andesite", &idata.Config)

	if idata.Config.Version == 0 {
		idata.Config.Version = 1
	}
	Log("Using config version:", idata.Config.Version)

	if idata.Config.Version != idata.RequiredConfigVersion {
		DieOnError(
			E(F("Current idata.Config.json version '%d' does not match required version '%d'.", idata.Config.Version, idata.RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", idata.RequiredConfigVersion),
		)
	}

	//

	etc.MFS.Add(http.Dir("./www/"))

	statikFS, err := fs.New()
	DieOnError(err)
	etc.MFS.Add(http.FileSystem(statikFS))

	//

	idata.Config.Port = iutil.FindFirstNonZero(*flagPort, idata.Config.Port, 8000)
	Log("Discovered option:", "--port", idata.Config.Port)
	idata.Config.HTTPBase = iutil.FindFirstNonEmpty(*flagBase, idata.Config.HTTPBase, "/")
	Log("Discovered option:", "--base", idata.Config.HTTPBase)
	idata.Config.Root = iutil.FindFirstNonEmpty(*flagRoot, idata.Config.Root)
	Log("Discovered option:", "--root", idata.Config.Root)
	idata.Config.Public = iutil.FindFirstNonEmpty(*flagPublic, idata.Config.Public)
	Log("Discovered option:", "--public", idata.Config.Public)

	if *flagSearch {
		idata.Config.SearchOn = true
	}

	//
	// configure root dir

	idata.Config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(idata.Config.Root, "~", idata.HomedirPath, -1)))
	Log("Sharing private files from " + idata.Config.Root)
	DieOnError(Assert(DoesDirectoryExist(idata.Config.Root), "Please pass a valid directory as a root parameter!"))

	if len(idata.Config.Public) > 0 {
		idata.Config.Public, _ = filepath.Abs(idata.Config.Public)
		Log("Sharing public files from", idata.Config.Public)
		DieOnError(Assert(DoesDirectoryExist(idata.Config.Public), "Public root directory does not exist. Aborting!"))
	}

	//
	// add custom providers to the registry

	for _, item := range idata.Config.Providers {
		Log(1, item)
		oauth2.ProviderIDMap[item.ID] = item
	}

	//
	// database initialization

	etc.Database.CreateTableStruct("users", itypes.UserRow{})
	etc.Database.CreateTableStruct("access", itypes.UserAccessRow{})
	etc.Database.CreateTableStruct("shares", itypes.ShareRow{})
	etc.Database.CreateTableStruct("shares_discord_role", itypes.DiscordRoleAccessRow{})

	//
	// database upgrade (removing db prefixes in favor of provider column)

	prefixes := map[string]string{
		"reddit":    "1:",
		"github":    "2:",
		"google":    "3:",
		"facebook":  "4:",
		"microsoft": "5:",
	}
	for _, item := range iutil.QueryAllUsers() {
		for k, v := range prefixes {
			if strings.HasPrefix(item.Snowflake, v) {
				sn := item.Snowflake[len(v):]
				tid := strconv.Itoa(item.ID)
				Log("[db-upgrade]", item.Snowflake, "is now", sn, "as", k)
				etc.Database.QueryDoUpdate("users", "snowflake", sn, "id", tid)
				etc.Database.QueryDoUpdate("users", "provider", k, "id", tid)
			}
		}
	}

	//
	// graceful stop

	etc.RunOnClose(func() {
		Log("Gracefully shutting down...")

		Log("Saving database to disk")
		etc.Database.Close()

		if idata.Config.SearchOn {
			Log("Closing filesystem watcher")
			watcher.Close()
		}

		Log("Done!")
	})

	//
	// initialize filesystem watching

	if idata.Config.SearchOn {
		go initFsWatcher()
	}

	//
	// handlebars helpers

	raymond.RegisterHelper("url_name", func(x string) string {
		return strings.Replace(url.PathEscape(x), "%2F", "/", -1)
	})

	//
	// http server setup and launch

	mw := iutil.ChainMiddleware(iutil.MwAddAttribution)

	http.HandleFunc("/", mw(http.FileServer(etc.MFS).ServeHTTP))
	http.HandleFunc("/login", mw(oauth2.HandleMultiOAuthLogin(helperIsLoggedIn, "./files/", idata.Config.Clients)))
	http.HandleFunc("/callback", mw(oauth2.HandleMultiOAuthCallback("./files/", idata.Config.Clients, helperOA2SaveInfo)))
	http.HandleFunc("/test", mw(handleTest))
	http.HandleFunc("/files/", mw(handleDirectoryListing(handleFileListing)))
	http.HandleFunc("/admin", mw(handleAdmin))
	http.HandleFunc("/api/access/delete", mw(handleAccessDelete))
	http.HandleFunc("/api/access/update", mw(handleAccessUpdate))
	http.HandleFunc("/api/access/create", mw(handleAccessCreate))
	http.HandleFunc("/open/", mw(handleDirectoryListing(handleShareListing)))
	http.HandleFunc("/api/share/create", mw(handleShareCreate))
	http.HandleFunc("/api/share/update", mw(handleShareUpdate))
	http.HandleFunc("/api/share/delete", mw(handleShareDelete))
	http.HandleFunc("/logout", mw(handleLogout))
	http.HandleFunc("/search", mw(handleSearch))
	http.HandleFunc("/api/search", mw(handleSearchAPI))
	http.HandleFunc("/public/", mw(handleDirectoryListing(handlePublicListing)))
	http.HandleFunc("/api/access_discord_role/create", mw(handleDiscordRoleAccessCreate))
	http.HandleFunc("/api/access_discord_role/update", mw(handleDiscordRoleAccessUpdate))
	http.HandleFunc("/api/access_discord_role/delete", mw(handleDiscordRoleAccessDelete))
	http.HandleFunc("/regen_passkey", mw(handleRegenPasskey))

	if !IsPortAvailable(idata.Config.Port) {
		DieOnError(
			E(F("Binding to port %d failed.", idata.Config.Port)),
			"It may be taken or you may not have permission to. Aborting!",
		)
		return
	}

	p := strconv.Itoa(idata.Config.Port)
	Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}
