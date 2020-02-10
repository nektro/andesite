package main

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/handler"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"
	"github.com/nektro/andesite/search"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
	"github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)

var (
	Version = "vMASTER"
)

func main() {
	idata.Version = Version
	Log("Initializing Andesite " + idata.Version + "...")

	pflag.IntVar(&idata.Config.Version, "version", idata.RequiredConfigVersion, "Config version to use.")
	pflag.StringVar(&idata.Config.Root, "root", "", "Path of root directory for files")
	pflag.IntVar(&idata.Config.Port, "port", 8000, "Port to open server on")
	pflag.StringVar(&idata.Config.HTTPBase, "base", "/", "Http Origin Path")
	pflag.StringVar(&idata.Config.Public, "public", "", "Public root of files to serve")
	pflag.BoolVar(&idata.Config.SearchOn, "enable-search", false, "Set to true to enable search database")
	flagDGS := pflag.String("discord-guild-id", "", "")
	flagDBT := pflag.String("discord-bot-token", "", "")
	etc.PreInit("andesite")

	etc.Init("andesite", &idata.Config, "./files/", helperOA2SaveInfo)

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

	//

	if idata.Config.Version == 0 {
		idata.Config.Version = 1
	}
	if idata.Config.Version != idata.RequiredConfigVersion {
		DieOnError(
			E(F("Current idata.Config.json version '%d' does not match required version '%d'.", idata.Config.Version, idata.RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", idata.RequiredConfigVersion),
		)
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
			search.Close()
		}

		Log("Done!")
	})

	//
	// initialize filesystem watching

	if idata.Config.SearchOn {
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

	http.HandleFunc("/test", iutil.Mw(handler.HandleTest))

	if len(idata.Config.Root) > 0 {
		idata.Config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(idata.Config.Root, "~", idata.HomedirPath, -1)))
		Log("Sharing private files from " + idata.Config.Root)
		DieOnError(Assert(DoesDirectoryExist(idata.Config.Root), "Please pass a valid directory as a root parameter!"))
		idata.DataPaths["files"] = idata.Config.Root

		http.HandleFunc("/admin", iutil.Mw(handler.HandleAdmin))
		http.HandleFunc("/admin/users", iutil.Mw(handler.HandleAdminUsers))

		http.HandleFunc("/api/access/delete", iutil.Mw(handler.HandleAccessDelete))
		http.HandleFunc("/api/access/update", iutil.Mw(handler.HandleAccessUpdate))
		http.HandleFunc("/api/access/create", iutil.Mw(handler.HandleAccessCreate))

		http.HandleFunc("/api/share/create", iutil.Mw(handler.HandleShareCreate))
		http.HandleFunc("/api/share/update", iutil.Mw(handler.HandleShareUpdate))
		http.HandleFunc("/api/share/delete", iutil.Mw(handler.HandleShareDelete))

		http.HandleFunc("/api/access_discord_role/create", iutil.Mw(handler.HandleDiscordRoleAccessCreate))
		http.HandleFunc("/api/access_discord_role/update", iutil.Mw(handler.HandleDiscordRoleAccessUpdate))
		http.HandleFunc("/api/access_discord_role/delete", iutil.Mw(handler.HandleDiscordRoleAccessDelete))

		http.HandleFunc("/regen_passkey", iutil.Mw(handler.HandleRegenPasskey))
		http.HandleFunc("/logout", iutil.Mw(handler.HandleLogout))

		http.HandleFunc("/files/", iutil.Mw(handler.HandleDirectoryListing(handler.HandleFileListing)))
		http.HandleFunc("/open/", iutil.Mw(handler.HandleDirectoryListing(handler.HandleShareListing)))
	}

	if len(idata.Config.Public) > 0 {
		idata.Config.Public, _ = filepath.Abs(idata.Config.Public)
		Log("Sharing public files from", idata.Config.Public)
		DieOnError(Assert(DoesDirectoryExist(idata.Config.Public), "Public root directory does not exist. Aborting!"))
		idata.DataPaths["public"] = idata.Config.Public

		http.HandleFunc("/public/", iutil.Mw(handler.HandleDirectoryListing(handler.HandlePublicListing)))
	}

	etc.StartServer(idata.Config.Port)
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
	iutil.QueryAssertUserName(provider, id, name)
	Log("[user-login]", provider, id, name)
}
