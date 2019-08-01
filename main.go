package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	"github.com/aymerick/raymond"
	"github.com/gorilla/sessions"
	"github.com/mitchellh/go-homedir"
	"github.com/nektro/go-util/logger"
	"github.com/nektro/go-util/sqlite"
	"github.com/nektro/go-util/types"
	"github.com/nektro/go.etc"
	"github.com/nektro/go.oauth2"
	"github.com/rakyll/statik/fs"
	flag "github.com/spf13/pflag"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)

const (
	version     = 1
	accessToken = "access_token"
)

func main() {
	log.Log(logger.LevelINFO, "Initializing Andesite...")

	flagRoot := flag.String("root", "", "Path of root directory for files")
	flagPort := flag.Int("port", 0, "Port to open server on")
	flagAdmin := flag.String("admin", "", "Discord User ID of the user that is distinguished as a site owner")
	flagTheme := flag.StringArray("theme", []string{}, "Name of the custom theme to use for the HTML pages")
	flagBase := flag.String("base", "", "Http Origin Path")
	flagLLevel := flag.Int("log-level", int(logger.LevelINFO), "Logging level to be used for github.com/nektro/go-util/logger")
	flagPublic := flag.String("public", "", "Public root of files to serve")
	flagSearch := flag.Bool("enable-search", false, "Set to true to enable search database")
	flag.Parse()

	//
	// parse options and find config

	log.Level = logger.LogLevel(*flagLLevel)
	homedir, _ := homedir.Dir()

	metaDir := homedir + "/.config/andesite"
	configPath := metaDir + "/config.json"
	log.Log(logger.LevelINFO, "Reading configuration info from", configPath)

	if !DoesFileExist(configPath) {
		log.Log(logger.LevelDEBUG, "Configuration file does not exist, creating blank!")
		os.MkdirAll(metaDir, os.ModePerm)
		ioutil.WriteFile(configPath, []byte("{}"), os.ModePerm)
	}
	etc.InitConfig(configPath, &config)

	config.Root = findFirstNonEmpty(*flagRoot, config.Root)
	log.Log(logger.LevelDEBUG, "Discovered option:", "--root", config.Root)
	config.Port = findFirstNonZero(*flagPort, config.Port, 8000)
	log.Log(logger.LevelDEBUG, "Discovered option:", "--port", config.Port)
	config.HTTPBase = findFirstNonEmpty(*flagBase, config.HTTPBase, "/")
	log.Log(logger.LevelDEBUG, "Discovered option:", "--base", config.HTTPBase)
	config.Public = findFirstNonEmpty(*flagPublic, config.Public)
	log.Log(logger.LevelDEBUG, "Discovered option:", "--public", config.Public)

	if *flagSearch {
		config.SearchOn = true
	}

	//
	// configure root dir

	config.Root, _ = filepath.Abs(filepath.Clean(strings.Replace(config.Root, "~", homedir, -1)))
	log.Log(logger.LevelINFO, "Sharing private files from "+config.Root)
	DieOnError(Assert(DoesDirectoryExist(config.Root), "Please pass a valid directory as a root parameter!"))

	if len(config.Public) > 0 {
		config.Public, _ = filepath.Abs(config.Public)
		log.Log(logger.LevelINFO, "Sharing public files from", config.Public)
		DieOnError(Assert(DoesDirectoryExist(config.Public), "Public root directory does not exist. Aborting!"))
	}

	//
	// discover OAuth2 config info

	if len(config.Auth) == 0 {
		config.Auth = "discord"
	}
	if cfp, ok := Oauth2Providers[config.Auth]; ok {
		cidp := findStructValueWithTag(&config, "json", config.Auth).Interface().(*ConfigIDP)
		DieOnError(Assert(cidp != nil, F("Authorization keys not set for identity prodvider '%s' in config.json!", config.Auth)))
		DieOnError(Assert(cidp.ID != "", F("App ID not set for identity prodvider '%s' in config.json!", config.Auth)))
		DieOnError(Assert(cidp.Secret != "", F("App Secret not set for identity prodvider '%s' in config.json!", config.Auth)))
		oauth2AppConfig = cidp
		oauth2Provider = cfp
	} else {
		foundP := false
		for _, item := range config.Providers {
			if item.ID == config.Auth {
				oauth2Provider = Oauth2Provider{item, config.Auth}
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
			if item.Auth == config.Auth {
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

	database = sqlite.Connect(metaDir)
	checkErr(database.Ping())

	database.CreateTable("users", []string{"id", "int primary key"}, [][]string{
		{"snowflake", "text"},
		{"admin", "tinyint(1)"},
		{"name", "text"},
	})
	database.CreateTable("access", []string{"id", "int primary key"}, [][]string{
		{"user", "int"},
		{"path", "text"},
	})
	database.CreateTable("shares", []string{"id", "int primary key"}, [][]string{
		{"hash", "text"}, // character(32)
		{"path", "text"},
	})

	//
	// admin creation from (optional) CLI argument

	if *flagAdmin != "" {
		uu, ok := queryUserBySnowflake(*flagAdmin)
		if !ok {
			uid := database.QueryNextID("users")
			queryDoAddUser(uid, *flagAdmin, true, "")
			log.Log(logger.LevelINFO, F("Added user %s as an admin", *flagAdmin))
		} else {
			if !uu.Admin {
				database.QueryDoUpdate("users", "admin", "1", "id", strconv.FormatInt(int64(uu.ID), 10))
				log.Log(logger.LevelINFO, F("Set user '%s's status to admin", uu.Snowflake))
			}
		}
		nu, _ := queryUserBySnowflake(*flagAdmin)
		if !Contains(queryAccess(nu), "/") {
			aid := database.QueryNextID("access")
			database.Query(true, F("insert into access values ('%d', '%d', '/')", aid, nu.ID))
			log.Log(logger.LevelINFO, F("Gave %s root folder access", nu.Name))
		}
	}

	//
	// graceful stop

	gracefulStop := make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	go func() {
		sig := <-gracefulStop
		log.Log(logger.LevelINFO, F("Caught signal '%+v'", sig))
		log.Log(logger.LevelINFO, "Gracefully shutting down...")

		log.Log(logger.LevelINFO, "Saving database to disk")
		database.Close()

		if config.SearchOn {
			log.Log(logger.LevelINFO, "Closing filesystem watcher")
			watcher.Close()
		}

		log.Log(logger.LevelINFO, "Done!")
		os.Exit(0)
	}()

	//
	// initialize filesystem watching

	if config.SearchOn {
		go initFsWatcher()
	}

	//
	// http server pre-setup

	etc.SetSessionName("session_andesite")
	p := strconv.Itoa(config.Port)
	dirs := []http.FileSystem{}

	//
	// theme setup

	for _, item := range *flagTheme {
		loc := metaDir + "/themes/" + item
		DieOnError(Assert(DoesDirectoryExist(loc), F("'%s' does not exist!", loc)))
		dirs = append(dirs, http.Dir(loc))
	}
	for _, item := range config.Themes {
		loc := metaDir + "/themes/" + item
		DieOnError(Assert(DoesDirectoryExist(loc), F("'%s' does not exist!", loc)))
		dirs = append(dirs, http.Dir(loc))
	}

	//
	// http server setup and launch

	statikFS, err := fs.New()
	if err != nil {
		log.Log(logger.LevelFATAL, err.Error())
		return
	}

	mw := chainMiddleware(mwAddAttribution)
	dirs = append(dirs, http.Dir("./www/"))
	dirs = append(dirs, http.FileSystem(statikFS))
	wwFFS = types.MultiplexFileSystem{dirs}

	http.HandleFunc("/", mw(http.FileServer(wwFFS).ServeHTTP))
	http.HandleFunc("/login", mw(oauth2.HandleOAuthLogin(helperIsLoggedIn, "./files/", oauth2Provider.IDP, oauth2AppConfig.ID)))
	http.HandleFunc("/callback", mw(oauth2.HandleOAuthCallback(oauth2Provider.IDP, oauth2AppConfig.ID, oauth2AppConfig.Secret, helperOA2SaveInfo, "./files")))
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

	if !IsPortAvailable(config.Port) {
		log.Log(logger.LevelFATAL, "Binding to port", config.Port, "failed. It may be taken or you may not have permission to. Aborting!")
		return
	}

	log.Log(logger.LevelINFO, "Initialization complete. Starting server on port "+p)
	http.ListenAndServe(":"+p, nil)
}

func readServerFile(path string) []byte {
	reader, _ := wwFFS.Open(path)
	bytes, _ := ioutil.ReadAll(reader)
	return bytes
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

func fullHost(r *http.Request) string {
	urL := "http"
	if r.TLS != nil {
		urL += "s"
	}
	return urL + "://" + r.Host
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
		me += F("%s%s (%s)", oauth2Provider.IDP.NamePrefix, sessName.(string), sessID.(string))
	}

	message := ""
	if fileOrAdmin {
		if showLogin {
			message = "You" + me + " do not have access to this resource."
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

func writeHandlebarsFile(r *http.Request, w http.ResponseWriter, file string, context map[string]interface{}) {
	template := string(readServerFile(file))
	result, _ := raymond.Render(template, context)
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintln(w, result)
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
	writeHandlebarsFile(r, w, "/response.hbs", map[string]interface{}{
		"title":   title,
		"message": message,
		"link":    link,
		"base":    config.HTTPBase,
	})
}

func containsAll(mp url.Values, keys ...string) bool {
	for _, item := range keys {
		if _, ok := mp[item]; !ok {
			return false
		}
	}
	return true
}

func apiBootstrapRequireLogin(r *http.Request, w http.ResponseWriter, method string, requireAdmin bool) (*sessions.Session, UserRow, error) {
	if r.Method != method {
		writeAPIResponse(r, w, false, "This action requires using HTTP "+method)
		return nil, UserRow{}, E("")
	}

	sess := etc.GetSession(r)
	sessID := sess.Values["user"]

	if sessID == nil {
		writeUserDenied(r, w, true, true)
		return nil, UserRow{}, E("")
	}

	userID := sessID.(string)
	user, ok := queryUserBySnowflake(userID)

	if !ok {
		writeResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+userID+")", "")
		return nil, UserRow{}, E("")
	}
	if requireAdmin && !user.Admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator. ("+userID+")")
		return nil, UserRow{}, E("")
	}

	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
		return nil, UserRow{}, E("")
	}

	return sess, user, nil
}

func doHttpRequest(req *http.Request) []byte {
	client := &http.Client{}
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return body
}

// @from https://gist.github.com/gbbr/fa652db0bab132976620bcb7809fd89a
func chainMiddleware(mw ...Middleware) Middleware {
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
