package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"syscall"

	"github.com/aymerick/raymond"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/nektro/go-util/types"
	"github.com/nektro/go-util/util"

	flag "github.com/spf13/pflag"

	_ "github.com/mattn/go-sqlite3"
)

const (
	version     = 1
	accessToken = "access_token"
	pthAnd      = "/.andesite/"
)

var (
	oauth2AppID     string
	oauth2AppSecret string
	oauth2Provider  Oauth2Provider
	database        *sql.DB
	wwFFS           types.MultiplexFileSystem
	httpBase        string
	rootDir         RootDir
	metaDir         string
	randomKey       = securecookie.GenerateRandomKey(32)
	store           = sessions.NewCookieStore(randomKey)
)

func main() {
	util.Log("Initializing Andesite...")

	flagRoot := flag.String("root", "", "Path of root directory for files")
	flagPort := flag.Int("port", 8000, "Port to open server on")
	flagAdmin := flag.String("admin", "", "Discord User ID of the user that is distinguished as the site owner")
	flagTheme := flag.String("theme", "", "Name of the custom theme to use for the HTML pages")
	flagBase := flag.String("base", "/", "")
	flagRType := flag.String("root-type", "dir", "Type of path --root points to. One of 'dir', 'http'")
	// flagMeta := flag.String("meta", "", "")
	flag.Parse()

	//
	// configure root dir

	switch RootDirType(*flagRType) {
	case RootTypeDir:
		rootDir = FsRoot{*flagRoot}
		s, _ := filepath.Abs(*flagRoot)
		dieOnError(assert(fileExists(s), "Please pass a valid directory as a --root parameter!"))
		metaDir = s + pthAnd
	// case RootTypeHttp:
	// 	rootDir = HttpRoot{*flagRoot}
	// 	s, _ := filepath.Abs(*flagMeta)
	// 	metaDir = s
	default:
		dieOnError(errors.New("Invalid root type"))
	}
	dieOnError(assert(fileExists(metaDir), ".andesite folder does not exist!"))
	util.Log("Starting in " + rootDir.Base())

	//
	// discover OAuth2 config info

	configPath := metaDir + "/config.json"
	dieOnError(assert(fileExists(configPath), "config.json does not exist!"))
	configBytes := readFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(configBytes, &config)

	ca := config["auth"]
	if ca == nil {
		dieOnError(errors.New("config.json[auth] is missing"))
	}
	cas := ca.(string)
	if len(cas) == 0 {
		cas = "discord"
	}
	if _, ok := Oauth2Providers[cas]; ok {
		if config[cas] == nil {
			dieOnError(errors.New(fmt.Sprintf("config.json[%s] is missing", cas)))
		}
		acm := config[cas].(map[string]interface{})
		oauth2AppID = acm["id"].(string)
		oauth2AppSecret = acm["secret"].(string)
	} else {
		dieOnError(errors.New(fmt.Sprintf("Invalid OAuth2 Client type '%s'", cas)))
	}

	oauth2Provider = Oauth2Providers[cas]

	//
	// database initialization

	db, err := sql.Open("sqlite3", "file:"+metaDir+"/access.db?mode=rwc&cache=shared")
	checkErr(err)
	database = db

	checkErr(database.Ping())

	createTable("users", []string{"id", "int primary key"}, [][]string{
		{"snowflake", "text"},
		{"admin", "tinyint(1)"},
		{"name", "text"},
	})
	createTable("access", []string{"id", "int primary key"}, [][]string{
		{"user", "int"},
		{"path", "text"},
	})
	createTable("shares", []string{"id", "int primary key"}, [][]string{
		{"hash", "text"}, // character(32)
		{"path", "text"},
	})

	//
	// admin creation from (optional) CLI argument

	if *flagAdmin != "" {
		uu, ok := queryUserBySnowflake(*flagAdmin)
		if !ok {
			uid := queryLastID("users") + 1
			aid := queryLastID("access") + 1
			queryDoAddUser(uid, *flagAdmin, true, "")
			query(fmt.Sprintf("insert into access values ('%d', '%d', '/')", aid, uid), true)
			util.Log(fmt.Sprintf("Added user %s as an admin", *flagAdmin))
		} else {
			if !uu.admin {
				queryDoUpdate("users", "admin", "1", "id", strconv.FormatInt(int64(uu.id), 10))
				util.Log(fmt.Sprintf("Set user '%s's status to admin", uu.snowflake))
			}
		}
	}

	//
	// theme check from (optional) CLI argument

	themeRootPath := ""
	themeDirName := ""
	if *flagTheme != "" {
		stheme := *flagTheme
		themeDirName = "theme-" + stheme
		themeRootPath = metaDir + themeDirName + "/"
		fi, err := os.Stat(themeRootPath)
		dieOnError(err, "Theme directory must exist if the --theme option is present")
		dieOnError(assert(fi.IsDir(), "Theme directory must be a directory!"))
	}

	//
	// set HTTP base dir
	httpBase = *flagBase

	//
	// graceful stop

	gracefulStop := make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	go func() {
		sig := <-gracefulStop
		util.Log(fmt.Sprintf("Caught signal '%+v'", sig))
		util.Log("Gracefully shutting down...")

		database.Close()
		util.Log("Saved database to disk")

		os.Exit(0)
	}()

	//

	p := strconv.Itoa(*flagPort)
	dirs := []http.FileSystem{}

	if themeRootPath != "" {
		dirs = append(dirs, http.Dir(themeRootPath))
	}

	mw := chainMiddleware(mwAddAttribution)
	dirs = append(dirs, http.Dir("www"))
	dirs = append(dirs, packr.New("", "./www/"))
	wwFFS = types.MultiplexFileSystem{dirs}

	http.HandleFunc("/", mw(http.FileServer(wwFFS).ServeHTTP))
	http.HandleFunc("/login", mw(handleOAuthLogin))
	http.HandleFunc("/callback", mw(handleOAuthCallback))
	http.HandleFunc("/token", mw(handleOAuthToken))
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

	util.Log("Initialization complete. Starting server on port " + p)
	http.ListenAndServe(":"+p, nil)
}

func dieOnError(err error, args ...string) {
	if err != nil {
		logError(fmt.Sprintf("%q", err))
		for _, item := range args {
			logError(item)
		}
		os.Exit(1)
	}
}

func assert(condition bool, errorMessage string) error {
	if condition {
		return nil
	}
	return errors.New(errorMessage)
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func logError(message string) {
	fmt.Println("[" + util.GetIsoDateTime() + "][error] " + message)
}

func readFile(path string) []byte {
	reader, _ := os.Open(path)
	bytes, _ := ioutil.ReadAll(reader)
	return bytes
}

func readServerFile(path string) []byte {
	reader, _ := wwFFS.Open(path)
	bytes, _ := ioutil.ReadAll(reader)
	return bytes
}

func reduceNumber(input int64, unit int64, base string, prefixes string) string {
	if input < unit {
		return fmt.Sprintf("%d "+base, input)
	}
	div, exp := int64(unit), 0
	for n := input / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ci", float64(input)/float64(div), prefixes[exp]) + base
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

func contains(stack []string, needle string) bool {
	for _, varr := range stack {
		if varr == needle {
			return true
		}
	}
	return false
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
		fmt.Println(fmt.Sprintf("%q: %s", err, args))
		debug.PrintStack()
	}
}

func writeUserDenied(r *http.Request, w http.ResponseWriter, fileOrAdmin bool, showLogin bool) {
	me := ""
	sess := getSession(r)
	sessName := sess.Values["name"]
	if sessName != nil {
		sessID := sess.Values["user"]
		me += fmt.Sprintf(" (%s%s - %s)", oauth2Provider.namePrefix, sessName.(string), sessID.(string))
	}

	message := ""
	if fileOrAdmin {
		if showLogin {
			message = "You" + me + " do not have access to this resource."
		} else {
			message = "Unable to find the requested resource for you."
		}
	} else {
		message = "Admin priviledge required. Access denied."
	}

	linkmsg := ""
	if showLogin {
		linkmsg = "Please <a href='" + httpBase + "login'>Log In</a>."
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
	writeResponse(r, w, titlemsg, message, "Return to <a href='"+httpBase+"admin'>the dashboard</a>.")
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

func getSession(r *http.Request) *sessions.Session {
	sess, _ := store.Get(r, "session_andesite")
	return sess
}

func writeResponse(r *http.Request, w http.ResponseWriter, title string, message string, link string) {
	writeHandlebarsFile(r, w, "/response.hbs", map[string]interface{}{
		"title":   title,
		"message": message,
		"link":    link,
		"base":    httpBase,
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
		return nil, UserRow{}, errors.New("")
	}

	sess := getSession(r)
	sessID := sess.Values["user"]

	if sessID == nil {
		writeUserDenied(r, w, true, true)
		return nil, UserRow{}, errors.New("")
	}

	userID := sessID.(string)
	user, ok := queryUserBySnowflake(userID)

	if !ok {
		writeResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+userID+")", "")
		return nil, UserRow{}, errors.New("")
	}
	if requireAdmin && !user.admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator. ("+userID+")")
		return nil, UserRow{}, errors.New("")
	}

	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
		return nil, UserRow{}, errors.New("")
	}

	return sess, user, nil
}

func doHttpRequest(req *http.Request) []byte {
	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
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
