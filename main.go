package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/gorilla/sessions"
	discord "github.com/nektro/go.discord"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/rakyll/statik/fs"
	flag "github.com/spf13/pflag"

	"github.com/nektro/andesite/internal/idata"
	"github.com/nektro/andesite/internal/itypes"

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

	idata.Config.Port = findFirstNonZero(*flagPort, idata.Config.Port, 8000)
	Log("Discovered option:", "--port", idata.Config.Port)
	idata.Config.HTTPBase = findFirstNonEmpty(*flagBase, idata.Config.HTTPBase, "/")
	Log("Discovered option:", "--base", idata.Config.HTTPBase)
	idata.Config.Root = findFirstNonEmpty(*flagRoot, idata.Config.Root)
	Log("Discovered option:", "--root", idata.Config.Root)
	idata.Config.Public = findFirstNonEmpty(*flagPublic, idata.Config.Public)
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
	for _, item := range queryAllUsers() {
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

	mw := chainMiddleware(mwAddAttribution)

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

func reduceNumber(input int64, unit int64, base string, prefixes string) string {
	if input < unit {
		return F("%d "+base, input)
	}
	div, exp := int64(unit), 0
	for n := input / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return F("%.1f %ci", float64(input)/float64(div), prefixes[exp]) + base
}

func byteCountIEC(b int64) string {
	return reduceNumber(b, 1024, "B", "KMGTPEZY")
}

func filter(stack []os.FileInfo, cb func(os.FileInfo) bool) []os.FileInfo {
	result := []os.FileInfo{}
	for _, item := range stack {
		if cb(item) {
			result = append(result, item)
		}
	}
	return result
}

func checkErr(err error, args ...string) {
	if err != nil {
		fmt.Println("Error")
		fmt.Println(F("%q: %s", err, args))
		debug.PrintStack()
	}
}

func writeUserDenied(r *http.Request, w http.ResponseWriter, fileOrAdmin bool, showLogin bool) {
	me := ""
	sess := etc.GetSession(r)
	sessName := sess.Values["name"]
	if sessName != nil {
		sessID := sess.Values["user"].(string)
		provider := sess.Values["provider"].(string)
		me += F("%s%s (%s)", oauth2.ProviderIDMap[provider].NamePrefix, sessName.(string), sessID)
	}

	message := ""
	if fileOrAdmin {
		if showLogin {
			message = "You " + me + " do not have access to this resource."
		} else {
			message = "Unable to find the requested resource for you " + me + "."
		}
	} else {
		message = "Admin priviledge required. Access denied."
	}

	linkmsg := ""
	if showLogin {
		linkmsg = "Please <a href='" + idata.Config.HTTPBase + "login'>Log In</a>."
		w.WriteHeader(http.StatusForbidden)
		writeResponse(r, w, "Forbidden", message, linkmsg)
	} else {
		w.WriteHeader(http.StatusForbidden)
		writeResponse(r, w, "Not Found", message, linkmsg)
	}
}

func writeAPIResponse(r *http.Request, w http.ResponseWriter, good bool, message string) {
	if !good {
		w.WriteHeader(http.StatusForbidden)
	}
	titlemsg := ""
	if good {
		titlemsg = "Update Successful"
	} else {
		titlemsg = "Update Failed"
	}
	writeResponse(r, w, titlemsg, message, "Return to <a href='"+idata.Config.HTTPBase+"admin'>the dashboard</a>.")
}

func boolToString(x bool) string {
	if x {
		return "1"
	}
	return "0"
}

func writeResponse(r *http.Request, w http.ResponseWriter, title string, message string, link string) {
	etc.WriteHandlebarsFile(r, w, "/response.hbs", map[string]interface{}{
		"title":   title,
		"message": message,
		"link":    link,
		"base":    idata.Config.HTTPBase,
	})
}

func writeLinkResponse(r *http.Request, w http.ResponseWriter, title string, message string, linkText string, href string) {
	writeResponse(r, w, title, message, "<a href=\""+href+"\">"+linkText+"</a>")
}

func containsAll(mp url.Values, keys ...string) bool {
	for _, item := range keys {
		if _, ok := mp[item]; !ok {
			return false
		}
	}
	return true
}

func apiBootstrapRequireLogin(r *http.Request, w http.ResponseWriter, method string, requireAdmin bool) (*sessions.Session, *itypes.UserRow, error) {
	if r.Method != method {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Add("Allow", "HEAD, "+method)
		writeAPIResponse(r, w, false, "This action requires using HTTP "+method)
		return nil, nil, E("")
	}

	sess := etc.GetSession(r)
	sessID := sess.Values["user"]

	if sessID == nil {
		pk := r.Header.Get("x-passkey")
		if len(pk) == 0 {
			writeUserDenied(r, w, true, true)
			return nil, nil, E("not logged in and no passkey found")
		}
		kq := etc.Database.QueryDoSelect("users", "passkey", pk)
		if !kq.Next() {
			writeUserDenied(r, w, true, true)
			return nil, nil, E("invalid passkey")
		}
		sessID = scanUser(kq).Snowflake
		kq.Close()
	}

	userID := sessID.(string)
	user, ok := queryUserBySnowflake(userID)

	if !ok {
		writeResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+userID+")", "")
		return nil, nil, E("")
	}
	if requireAdmin && !user.Admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator. ("+userID+")")
		return nil, nil, E("")
	}

	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
		return nil, nil, E("")
	}

	return sess, user, nil
}

// @from https://gist.github.com/gbbr/fa652db0bab132976620bcb7809fd89a
func chainMiddleware(mw ...itypes.Middleware) itypes.Middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i](last)
			}
			last(w, r)
		}
	}
}

func mwAddAttribution(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server", "nektro/andesite")
		next.ServeHTTP(w, r)
	}
}

func findFirstNonEmpty(values ...string) string {
	for _, item := range values {
		if len(item) > 0 {
			return item
		}
	}
	return ""
}

func findFirstNonZero(values ...int) int {
	for _, item := range values {
		if item != 0 {
			return item
		}
	}
	return 0
}

func writeJSON(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("content-type", "application/json")
	bytes, _ := json.Marshal(data)
	w.Write(bytes)
}

func generateNewUserPasskey(snowflake string) string {
	hash1 := md5.Sum([]byte(F("astheno.andesite.passkey.%s.%s", snowflake, T())))
	hash2 := hex.EncodeToString(hash1[:])
	return hash2[0:10]
}

func makeDiscordRequest(endpoint string, body url.Values) []byte {
	req, _ := http.NewRequest(http.MethodGet, idata.DiscordAPI+endpoint, strings.NewReader(body.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bot "+idata.Config.GetDiscordClient().Extra2)
	req.Header.Set("Accept", "application/json")
	res, _ := http.DefaultClient.Do(req)
	bys, _ := ioutil.ReadAll(res.Body)
	return bys
}

func fetchDiscordRole(guild string, role string) discord.GuildRole {
	bys := makeDiscordRequest("/guilds/"+guild+"/roles", url.Values{})
	roles := []discord.GuildRole{}
	json.Unmarshal(bys, &roles)
	for i, item := range roles {
		if item.ID == role {
			return roles[i]
		}
	}
	return discord.GuildRole{}
}

type DiscordGuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func fetchDiscordGuild(guild string) DiscordGuild {
	bys := makeDiscordRequest("/guilds/"+guild, url.Values{})
	var dg DiscordGuild
	json.Unmarshal(bys, &dg)
	return dg
}
