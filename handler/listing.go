package handler

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/util"
)

func HandleDirectoryListing(getAccess func(http.ResponseWriter, *http.Request) (string, string, []string, *db.UserRow, map[string]interface{}, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fileRoot, qpath, uAccess, user, extras, err := getAccess(w, r)

		w.Header().Add("access-control-allow-origin", "*")

		// if getAccess errored, response has already been written
		if err != nil {
			return
		}

		// disallow path tricks
		if strings.Contains(string(r.URL.Path), "..") {
			return
		}

		// disallow exploring dotfile folders
		if strings.Contains(qpath, "/.") {
			util.WriteUserDenied(r, w, true, false)
			return
		}

		// valid path check
		stat, err := os.Stat(fileRoot + qpath)
		if os.IsNotExist(err) {
			// 404
			util.WriteUserDenied(r, w, true, false)
			return
		}

		// server file/folder
		if stat.IsDir() {

			// ensure directory paths end in `/`
			if !strings.HasSuffix(r.URL.Path, "/") {
				w.Header().Set("Location", r.URL.Path+"/")
				w.WriteHeader(http.StatusFound)
				return
			}

			// get list of all files
			files, _ := ioutil.ReadDir(fileRoot + qpath)

			// hide dot files
			files = util.Filter(files, func(x os.FileInfo) bool {
				return !strings.HasPrefix(x.Name(), ".")
			})

			// amount of files in the directory
			l1 := len(files)

			// access check
			files = util.Filter(files, func(x os.FileInfo) bool {
				ok := false
				fpath := qpath + x.Name()
				for _, item := range uAccess {
					if strings.HasPrefix(item, fpath) || strings.HasPrefix(qpath, item) {
						ok = true
					}
				}
				return ok
			})

			// amount of files given access to
			l2 := len(files)

			if l1 > 0 && l2 == 0 {
				util.WriteUserDenied(r, w, true, false)
				return
			}

			data := make([]map[string]string, len(files))
			gi := 0
			for i := 0; i < len(files); i++ {
				name := files[i].Name()
				a := ""
				if files[i].IsDir() || files[i].Mode()&os.ModeSymlink != 0 {
					a = name + "/"
				} else {
					a = name
				}
				ext := filepath.Ext(a)
				if files[i].IsDir() {
					ext = ".folder"
				}
				if len(ext) == 0 {
					ext = ".asc"
				}
				data[gi] = map[string]string{
					"name":    a,
					"size":    ByteCountIEC(files[i].Size()),
					"mod":     files[i].ModTime().UTC().String()[:19],
					"ext":     ext[1:],
					"mod_raw": strconv.FormatInt(files[i].ModTime().UTC().Unix(), 10),
					"is_file": strconv.FormatBool(!files[i].IsDir()),
				}
				gi++
			}

			etc.WriteHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
				"version":   config.Version,
				"provider":  user.Provider,
				"user":      user,
				"path":      r.URL.Path[len(config.Config.HTTPBase)-1:],
				"files":     data,
				"admin":     user.Admin,
				"base":      config.Config.HTTPBase,
				"name":      oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
				"search_on": config.Config.SearchOn,
				"host":      FullHost(r),
				"extras":    extras,
			})
		} else {
			// access check
			can := false
			for _, item := range uAccess {
				if strings.HasPrefix(qpath, item) {
					can = true
				}
			}
			if can == false {
				util.WriteUserDenied(r, w, true, false)
				return
			}

			w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(qpath)))
			file, _ := os.Open(fileRoot + qpath)
			info, _ := os.Stat(fileRoot + qpath)
			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
		}
	}
}

// handler for http://andesite/files/*
func HandleFileListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *db.UserRow, map[string]interface{}, error) {
	_, user, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet, http.MethodHead}, false)
	if err != nil {
		return "", "", nil, nil, nil, err
	}
	u := strings.Split(r.URL.Path, "/")

	// get path
	// remove /files
	qpath := "/" + strings.Join(u[2:], "/")

	userAccess := db.QueryAccess(user)
	dc := config.Config.GetDiscordClient()

	if user.Provider == "discord" && dc.Extra1 != "" && dc.Extra2 != "" {
		dra := db.QueryAllDiscordRoleAccess()
		var p fastjson.Parser

		rurl := F("%s/guilds/%s/members/%s", config.DiscordAPI, dc.Extra1, user.Snowflake)
		req, _ := http.NewRequest(http.MethodGet, rurl, strings.NewReader(""))
		req.Header.Set("User-Agent", "nektro/andesite")
		req.Header.Set("Authorization", "Bot "+dc.Extra2)
		bys := DoHttpRequest(req)
		v, err := p.Parse(string(bys))
		if err != nil {
			fmt.Println(2, "err", err.Error())
		}
		if v != nil {
			for _, item := range dra {
				for _, i := range v.GetArray("roles") {
					s, _ := i.StringBytes()
					if string(s) == item.RoleID {
						userAccess = append(userAccess, item.Path)
					}
				}
			}
		}
	}
	userAccess = util.FilterStr(userAccess, func(s string) bool {
		return strings.HasPrefix(s, "/"+u[1]+"/") || s == "/"
	})
	userAccess = util.MapStr(userAccess, func(s string) string {
		if s == "/" {
			return s
		}
		return s[len(u[1])+1:]
	})

	return config.Config.Root, qpath, userAccess, user, map[string]interface{}{
		"user": user,
	}, nil
}

// handler for http://andesite/public/*
func HandlePublicListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *db.UserRow, map[string]interface{}, error) {
	// remove /public
	qpath := string(r.URL.Path[7:])
	qaccess := []string{}

	if len(config.Config.Public) > 0 {
		qaccess = append(qaccess, "/")
	}
	return config.Config.Public, qpath, qaccess, &db.UserRow{ID: -1, Name: "Guest", Provider: r.Host}, map[string]interface{}{}, nil
}
