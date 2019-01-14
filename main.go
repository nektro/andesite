package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/aymerick/raymond"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	_ "github.com/mattn/go-sqlite3"
)

const (
	version     = 1
	discordAPI  = "https://discordapp.com/api"
	accessToken = "access_token"
	pthAnd      = "/.andesite/"
)

var (
	oauth2AppID     string
	oauth2AppSecret string
	randomKey       = securecookie.GenerateRandomKey(32)
	store           = sessions.NewCookieStore(randomKey)
	database        *sql.DB
	wwFFS           FusingFileSystem
	httpBase        string
	rootDir         RootDir
	metaDir         string
)

func main() {
	flagRoot := flag.String("root", "", "Path of root directory for files")
	port := flag.Int("port", 8000, "Port to open server on")
	admin := flag.String("admin", "", "Discord User ID of the user that is distinguished as the site owner")
	theme := flag.String("theme", "", "Name of the custom theme to use for the HTML pages")
	flagBase := flag.String("base", "/", "")
	flagRType := flag.String("root-type", "dir", "Type of path -root points to. One of 'dir', 'http'")
	flagMeta := flag.String("meta", "", "")
	flag.Parse()

	//
	// configure root dir

	switch RootDirType(*flagRType) {
	case RootTypeDir:
		rootDir = FsRoot{*flagRoot}
		s, _ := filepath.Abs(*flagRoot)
		dieOnError(assert(fileExists(s), "Please pass a valid directory as a -root parameter!"))
		metaDir = s + pthAnd
	case RootTypeHttp:
		rootDir = HttpRoot{*flagRoot}
		s, _ := filepath.Abs(*flagMeta)
		metaDir = s
	default:
		dieOnError(errors.New("Invalid root type"))
	}
	dieOnError(assert(fileExists(metaDir), ".andesite folder does not exist!"))

	//

	log("Starting Andesite in " + rootDir.Base())

	//
	// discover OAuth2 config info

	configPath := metaDir + "/config.json"
	dieOnError(assert(fileExists(configPath), "config.json does not exist!"))
	configBytes := readFile(configPath)
	var config Config
	json.Unmarshal(configBytes, &config)

	if len(config.Auth) == 0 {
		config.Auth = "discord"
	}
	switch config.Auth {
	case "discord":
		oauth2AppID = config.Discord.ID
		oauth2AppSecret = config.Discord.Secret
	default:
		dieOnError(errors.New(fmt.Sprintf("Invalid OAuth2 Client type '%s'", config.Auth)))
	}

	//
	// database initialization

	db, err := sql.Open("sqlite3", "file:"+metaDir+"/access.db?mode=rwc&cache=shared")
	checkErr(err)
	database = db

	checkErr(database.Ping())

	createTable("users", []string{"id", "int primary key"}, [][]string{
		{"snowflake", "text"},
		{"admin", "tinyint(1)"},
	})
	createTable("access", []string{"id", "int primary key"}, [][]string{
		{"user", "int"},
		{"path", "text"},
	})

	//
	// admin creation from (optional) CLI argument

	if *admin != "" {
		uu, ok := queryUserBySnowflake(*admin)
		if !ok {
			uid := queryLastID("users") + 1
			aid := queryLastID("access") + 1
			query(fmt.Sprintf("insert into users values ('%d', '%s', '1')", uid, *admin), true)
			query(fmt.Sprintf("insert into access values ('%d', '%d', '/')", aid, uid), true)
			log(fmt.Sprintf("Added user %s as an admin", *admin))
		} else {
			if !uu.admin {
				query(fmt.Sprintf("update users set admin = '1' where id = '%d'", uu.id), true)
				log(fmt.Sprintf("Set user '%s's status to admin", uu.snowflake))
			}
		}
	}

	//
	// theme check from (optional) CLI argument

	themeRootPath := ""
	themeDirName := ""
	if *theme != "" {
		stheme := *theme
		themeDirName = "theme-" + stheme
		themeRootPath = metaDir + themeDirName + "/"
		fi, err := os.Stat(themeRootPath)
		dieOnError(err, "Theme directory must exist if the -theme option is present")
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
		log(fmt.Sprintf("Caught signal '%+v'", sig))
		log("Gracefully shutting down...")

		database.Close()
		log("Save database to disk")

		os.Exit(0)
	}()

	//

	mw := chainMiddleware(withAttribution)
	p := strconv.Itoa(*port)
	dirs := []http.FileSystem{}

	if themeRootPath != "" {
		dirs = append(dirs, http.Dir(themeRootPath))
	}

	dirs = append(dirs, http.Dir("www"))
	wwFFS = FusingFileSystem{dirs}

	http.HandleFunc("/", mw(http.FileServer(wwFFS).ServeHTTP))
	http.HandleFunc("/login", mw(handleOAuthLogin))
	http.HandleFunc("/callback", mw(handleOAuthCallback))
	http.HandleFunc("/token", mw(handleOAuthToken))
	http.HandleFunc("/test", mw(handleTest))
	http.HandleFunc("/files/", mw(handleFileListing))
	http.HandleFunc("/admin", mw(handleAdmin))
	http.HandleFunc("/api/access/delete", mw(handleAccessDelete))
	http.HandleFunc("/api/access/update", mw(handleAccessUpdate))
	http.HandleFunc("/api/access/create", mw(handleAccessCreate))

	log("Initialization complete. Starting server on port " + p)
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

func log(message string) {
	fmt.Println("[" + getIsoDateTime() + "][info]  " + message)
}

func logError(message string) {
	fmt.Println("[" + getIsoDateTime() + "][error] " + message)
}

func getIsoDateTime() string {
	vil := time.Now().UTC().String()
	return vil[0:19]
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

// @from https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func fullHost(r *http.Request) string {
	urL := "http"
	if r.TLS != nil {
		urL += "s"
	}
	return urL + "://" + r.Host
}

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, "andesite_session")
	return session
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

func writeUserDenied(w http.ResponseWriter, fileOrAdmin, showLogin bool) {
	w.WriteHeader(http.StatusForbidden)

	message := ""
	if fileOrAdmin {
		message = "You do not have access to this file/folder."
	} else {
		message = "Admin priviledge required. Access denied."
	}

	linkmsg := ""
	if showLogin {
		linkmsg = "Please <a href='" + httpBase + "login'>Log In</a>."
	}

	writeHandlebarsFile(w, "/response.hbs", map[string]interface{}{
		"title":   "Forbidden",
		"message": message,
		"link":    linkmsg,
		"base":    httpBase,
	})
}

func writeHandlebarsFile(w http.ResponseWriter, file string, context map[string]interface{}) {
	template := string(readServerFile(file))
	result, _ := raymond.Render(template, context)
	fmt.Fprintln(w, result)
}

func writeAPIResponse(w http.ResponseWriter, good bool, message string) {
	if !good {
		w.WriteHeader(http.StatusForbidden)
	}
	titlemsg := ""
	if good {
		titlemsg = "Update Successful"
	} else {
		titlemsg = "Update Failed"
	}
	writeHandlebarsFile(w, "/response.hbs", map[string]interface{}{
		"title":   titlemsg,
		"message": message,
		"link":    "Return to <a href='" + httpBase + "admin'>the dashboard</a>.",
		"base":    httpBase,
	})
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

func withAttribution(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server", "nektro/andesite")
		next.ServeHTTP(w, r)
	}
}
