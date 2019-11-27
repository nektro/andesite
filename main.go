package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
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

	etc.Init("andesite", &idata.Config, "./files/", helperOA2SaveInfo)

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

	http.HandleFunc("/test", iutil.Mw(handleTest))

	if len(idata.Config.Root) > 0 {
		idata.Config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(idata.Config.Root, "~", idata.HomedirPath, -1)))
		Log("Sharing private files from " + idata.Config.Root)
		DieOnError(Assert(DoesDirectoryExist(idata.Config.Root), "Please pass a valid directory as a root parameter!"))

		http.HandleFunc("/files/", iutil.Mw(handleDirectoryListing(handleFileListing)))
		http.HandleFunc("/admin", iutil.Mw(handleAdmin))
		http.HandleFunc("/api/access/delete", iutil.Mw(handleAccessDelete))
		http.HandleFunc("/api/access/update", iutil.Mw(handleAccessUpdate))
		http.HandleFunc("/api/access/create", iutil.Mw(handleAccessCreate))
		http.HandleFunc("/open/", iutil.Mw(handleDirectoryListing(handleShareListing)))
		http.HandleFunc("/api/share/create", iutil.Mw(handleShareCreate))
		http.HandleFunc("/api/share/update", iutil.Mw(handleShareUpdate))
		http.HandleFunc("/api/share/delete", iutil.Mw(handleShareDelete))
		http.HandleFunc("/logout", iutil.Mw(handleLogout))
		http.HandleFunc("/api/access_discord_role/create", iutil.Mw(handleDiscordRoleAccessCreate))
		http.HandleFunc("/api/access_discord_role/update", iutil.Mw(handleDiscordRoleAccessUpdate))
		http.HandleFunc("/api/access_discord_role/delete", iutil.Mw(handleDiscordRoleAccessDelete))
		http.HandleFunc("/regen_passkey", iutil.Mw(handleRegenPasskey))
		http.HandleFunc("/admin/users", iutil.Mw(handleAdminUsers))
	}

	if len(idata.Config.Public) > 0 {
		idata.Config.Public, _ = filepath.Abs(idata.Config.Public)
		Log("Sharing public files from", idata.Config.Public)
		DieOnError(Assert(DoesDirectoryExist(idata.Config.Public), "Public root directory does not exist. Aborting!"))

		http.HandleFunc("/public/", iutil.Mw(handleDirectoryListing(handlePublicListing)))
	}

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
