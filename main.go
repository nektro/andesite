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
	"reflect"
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

	etc.Init("andesite", &config)

	if config.Version == 0 {
		config.Version = 1
	}
	Log("Using config version:", config.Version)

	if config.Version != RequiredConfigVersion {
		DieOnError(
			E(F("Current config.json version '%d' does not match required version '%d'.", config.Version, RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", RequiredConfigVersion),
		)
	}

	//

	etc.MFS.Add(http.Dir("./www/"))

	statikFS, err := fs.New()
	DieOnError(err)
	etc.MFS.Add(http.FileSystem(statikFS))

	//

	config.Port = findFirstNonZero(*flagPort, config.Port, 8000)
	Log("Discovered option:", "--port", config.Port)
	config.HTTPBase = findFirstNonEmpty(*flagBase, config.HTTPBase, "/")
	Log("Discovered option:", "--base", config.HTTPBase)
	config.Root = findFirstNonEmpty(*flagRoot, config.Root)
	Log("Discovered option:", "--root", config.Root)
	config.Public = findFirstNonEmpty(*flagPublic, config.Public)
	Log("Discovered option:", "--public", config.Public)

	if *flagSearch {
		config.SearchOn = true
	}

	//
	// configure root dir

	config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(config.Root, "~", homedirPath, -1)))
	Log("Sharing private files from " + config.Root)
	DieOnError(Assert(DoesDirectoryExist(config.Root), "Please pass a valid directory as a root parameter!"))

	if len(config.Public) > 0 {
		config.Public, _ = filepath.Abs(config.Public)
		Log("Sharing public files from", config.Public)
		DieOnError(Assert(DoesDirectoryExist(config.Public), "Public root directory does not exist. Aborting!"))
	}

	//
	// discover OAuth2 config info

	if len(config.Auth) == 0 {
		config.Auth = "discord"
	}
	if cfp, ok := oauth2.ProviderIDMap[config.Auth]; ok {
		cidp := findStructValueWithTag(&config, "json", config.Auth).Interface().(*oauth2.AppConf)
		DieOnError(Assert(cidp != nil, F("Authorization keys not set for identity prodvider '%s' in config.json!", config.Auth)))
		DieOnError(Assert(cidp.ID != "", F("App ID not set for identity prodvider '%s' in config.json!", config.Auth)))
		DieOnError(Assert(cidp.Secret != "", F("App Secret not set for identity prodvider '%s' in config.json!", config.Auth)))
		oauth2AppConfig = cidp
		oauth2Provider = cfp
	} else {
		foundP := false
		for _, item := range config.Providers {
			if item.ID == config.Auth {
				oauth2Provider = item
				foundP = true
				break
			}
		}
		if !foundP {
			DieOnError(E(F("Unable to find OAuth2 app type '%s' in config.json", config.Auth)))
		}
		//
		foundI := false
		for _, item := range config.CustomIds {
			if item.For == config.Auth {
				oauth2AppConfig = &item
				foundI = true
				break
			}
		}
		if !foundI {
			DieOnError(E(F("Unable to find OAuth2 client config for '%s' config.json", config.Auth)))
		}
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

		if config.SearchOn {
			Log("Closing filesystem watcher")
			watcher.Close()
		}

		Log("Done!")
	})

	//
	// initialize filesystem watching

	if config.SearchOn {
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
	http.HandleFunc("/login", mw(oauth2.HandleOAuthLogin(helperIsLoggedIn, "./files/", oauth2Provider, oauth2AppConfig.ID)))
	http.HandleFunc("/callback", mw(oauth2.HandleOAuthCallback(oauth2Provider, oauth2AppConfig.ID, oauth2AppConfig.Secret, helperOA2SaveInfo, "./files")))


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

	if !IsPortAvailable(config.Port) {
		DieOnError(
			E(F("Binding to port %d failed.", config.Port)),
			"It may be taken or you may not have permission to. Aborting!",
		)
		return
	}

	p := strconv.Itoa(config.Port)
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
		sessID := sess.Values["user"]
		me += F("%s%s (%s)", oauth2Provider.NamePrefix, sessName.(string), sessID.(string))
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
		linkmsg = "Please <a href='" + config.HTTPBase + "login'>Log In</a>."
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
	writeResponse(r, w, titlemsg, message, "Return to <a href='"+config.HTTPBase+"admin'>the dashboard</a>.")
}

func fixID(id interface{}) string {
	switch id.(type) {
	case float64:
		return strconv.Itoa(int(id.(float64)))
	}
	return id.(string)
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
		"base":    config.HTTPBase,
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

func apiBootstrapRequireLogin(r *http.Request, w http.ResponseWriter, method string, requireAdmin bool) (*sessions.Session, itypes.UserRow, error) {
	if r.Method != method {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Add("Allow", "HEAD, "+method)
		writeAPIResponse(r, w, false, "This action requires using HTTP "+method)
		return nil, itypes.UserRow{}, E("")
	}

	sess := etc.GetSession(r)
	sessID := sess.Values["user"]

	if sessID == nil {
		pk := r.Header.Get("x-passkey")
		if len(pk) == 0 {
			writeUserDenied(r, w, true, true)
			return nil, itypes.UserRow{}, E("not logged in and no passkey found")
		}
		kq := etc.Database.QueryDoSelect("users", "passkey", pk)
		if !kq.Next() {
			writeUserDenied(r, w, true, true)
			return nil, itypes.UserRow{}, E("invalid passkey")
		}
		sessID = scanUser(kq).Snowflake
		kq.Close()
	}

	userID := sessID.(string)
	user, ok := queryUserBySnowflake(userID)

	if !ok {
		writeResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+userID+")", "")
		return nil, itypes.UserRow{}, E("")
	}
	if requireAdmin && !user.Admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator. ("+userID+")")
		return nil, itypes.UserRow{}, E("")
	}

	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
		return nil, itypes.UserRow{}, E("")
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

func findStructValueWithTag(item interface{}, ttype string, tag string) reflect.Value {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get(ttype) == tag {
			return v.FieldByName(f.Name)
		}
	}
	return reflect.Zero(nil)
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
	req, _ := http.NewRequest(http.MethodGet, DiscordAPI+endpoint, strings.NewReader(body.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bot "+config.GetDiscordClient().Extra2)
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
