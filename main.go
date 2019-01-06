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
	"strings"
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
)

var (
	dataRootPath     = ""
	discordAppID     = ""
	discordAppSecret = ""
	randomKey        = securecookie.GenerateRandomKey(32)
	store            = sessions.NewCookieStore(randomKey)
	database         *sql.DB
)

func main() {
	pathRaw := flag.String("root", "", "Path of root directory for files")
	port := flag.Int("port", 8000, "Port to open server on")
	admin := flag.String("admin", "", "Discord User ID of the user that is distinguished as the site owner")
	flag.Parse()

	dieOnError(assert(len(*pathRaw) > 1, "Please pass a directory as a -root parameter!"))
	if !strings.HasSuffix(*pathRaw, "/") && !strings.HasSuffix(*pathRaw, "\\") {
		*pathRaw += string(os.PathSeparator)
	}
	drp, _ := filepath.Abs(*pathRaw)
	dataRootPath = drp
	dieOnError(assert(fileExists(dataRootPath), "Path specified does not exist!"))

	log("Starting Andesite in " + dataRootPath)
	dieOnError(assert(fileExists(dataRootPath+"/.andesite/"), ".andesite folder does not exist!"))

	configPath := dataRootPath + "/.andesite/config.json"
	dieOnError(assert(fileExists(configPath), "config.json does not exist!"))
	configBytes := readFile(configPath)
	var config Config
	json.Unmarshal(configBytes, &config)
	discordAppID = config.Discord.ID
	discordAppSecret = config.Discord.Secret

	//
	// database initialization

	db, err := sql.Open("sqlite3", "file:"+dataRootPath+"/.andesite/"+"access.db?mode=rwc&cache=shared")
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

func dieOnError(e error) {
	if e != nil {
		logError(e.Error())
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
	session, _ := store.Get(r, "session")
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

func writeUserDenied(w http.ResponseWriter, message string, show_login bool) {
	w.WriteHeader(http.StatusForbidden)
	writeHandlebarsFile(w, "./www/denied.hbs", map[string]interface{}{
		"denial_message": message,
		"need_login":     show_login,
	})
}

func writeHandlebarsFile(w http.ResponseWriter, file string, context map[string]interface{}) {
	template := string(readFile(file))
	result, _ := raymond.Render(template, context)
	fmt.Fprintln(w, result)
}

func writeAPIResponse(w http.ResponseWriter, good bool, message string) {
	if !good {
		w.WriteHeader(http.StatusForbidden)
	}
	writeHandlebarsFile(w, "./www/response.hbs", map[string]interface{}{
		"good":    good,
		"bad":     !good,
		"message": message,
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
