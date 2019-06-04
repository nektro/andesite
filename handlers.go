package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/nektro/go-util/util"
	"github.com/nektro/go.etc"
)

func helperIsLoggedIn(r *http.Request) bool {
	sess := etc.GetSession(r)
	_, ok := sess.Values["user"]
	return ok
}

func helperOA2SaveInfo(w http.ResponseWriter, r *http.Request, provider string, id string, name string) {
	sess := etc.GetSession(r)
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Save(r, w)
	queryAssertUserName(id, name)
}

// handler for http://andesite/test
func handleTest(w http.ResponseWriter, r *http.Request) {
	// sessions test
	// increment number every refresh
	sess := etc.GetSession(r)
	i := sess.Values["int"]
	if i == nil {
		i = 0
	}
	j := i.(int)
	sess.Values["int"] = j + 1
	sess.Save(r, w)
	fmt.Fprintf(w, strconv.Itoa(j))
}

func handleDirectoryListing(getAccess func(http.ResponseWriter, *http.Request) (string, []string, string, string, bool, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		qpath, uAccess, uID, uName, isAdmin, err := getAccess(w, r)

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
			writeUserDenied(r, w, true, false)
			return
		}

		// valid path check
		stat, err := rootDir.Stat(qpath)
		if os.IsNotExist(err) {
			// 404
			writeUserDenied(r, w, true, false)
			return
		}

		// server file/folder
		if stat.IsDir() {
			w.Header().Add("Content-Type", "text/html")

			// get list of all files
			files, _ := rootDir.ReadDir(qpath)

			// hide dot files
			files = filter(files, func(x os.FileInfo) bool {
				return !strings.HasPrefix(x.Name(), ".")
			})

			// amount of files in the directory
			l1 := len(files)

			// access check
			files = filter(files, func(x os.FileInfo) bool {
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
				writeUserDenied(r, w, true, false)
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
					"name": a,
					"size": byteCountIEC(files[i].Size()),
					"mod":  files[i].ModTime().UTC().String()[:19],
					"ext":  ext[1:],
				}
				gi++
			}

			writeHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
				"user":  uID,
				"path":  qpath,
				"files": data,
				"admin": isAdmin,
				"base":  httpBase,
				"name":  oauth2Provider.idp.NamePrefix + uName,
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
				writeUserDenied(r, w, true, false)
				return
			}

			w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(qpath)))
			file, _ := rootDir.ReadFile(qpath)
			info, _ := rootDir.Stat(qpath)
			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
		}
	}
}

// handler for http://andesite/files/*
func handleFileListing(w http.ResponseWriter, r *http.Request) (string, []string, string, string, bool, error) {
	_, user, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if errr != nil {
		return "", []string{}, "", "", false, errors.New("")
	}

	// get path
	// remove /files
	qpath := string(r.URL.Path[6:])

	userUser, _ := queryUserBySnowflake(user.snowflake)
	userAccess := queryAccess(user)

	return qpath, userAccess, user.snowflake, user.name, userUser.admin, nil
}

// handler for http://andesite/admin
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, true)
	if errr != nil {
		return
	}

	//
	accesses := queryAllAccess()
	shares := queryAllShares()
	writeHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"user":     user.snowflake,
		"accesses": accesses,
		"base":     httpBase,
		"name":     oauth2Provider.idp.NamePrefix + user.name,
		"shares":   shares,
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "snowflake") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := r.PostForm.Get("id")
	iid, err := strconv.ParseInt(aid, 10, 32)
	if err != nil {
		writeAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	database.Query(true, fmt.Sprintf("delete from access where id = '%d'", iid))
	writeAPIResponse(r, w, true, fmt.Sprintf("Removed access from %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	aid := r.PostForm.Get("id")
	iid, err := strconv.ParseInt(string(aid), 10, 32)
	if err != nil {
		writeAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	if !containsAll(r.PostForm, "snowflake", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	queryDoUpdate("access", "path", r.PostForm.Get("path"), "id", strconv.FormatInt(iid, 10))
	writeAPIResponse(r, w, true, fmt.Sprintf("Updated access for %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "snowflake", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := database.QueryNextID("access")
	asn := r.PostForm.Get("snowflake")
	apt := r.PostForm.Get("path")
	//
	u, ok := queryUserBySnowflake(asn)
	aud := -1
	if ok {
		aud = u.id
	} else {
		aud = database.QueryNextID("users")
		queryDoAddUser(aud, asn, false, "")
	}
	//
	database.QueryPrepared(true, "insert into access values (?, ?, ?)", aid, aud, apt)
	writeAPIResponse(r, w, true, fmt.Sprintf("Created access for %s.", asn))
}

func handleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := database.QueryNextID("shares")
	ahs1 := md5.Sum([]byte(fmt.Sprintf("astheno.andesite.share.%s.%s", strconv.FormatInt(int64(aid), 10), util.GetIsoDateTime())))
	ahs2 := hex.EncodeToString(ahs1[:])
	fpath := r.PostForm.Get("path")
	//
	database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ahs2, fpath)
	writeAPIResponse(r, w, true, fmt.Sprintf("Created share with code %s for folder %s.", ahs2, fpath))
}

func handleShareListing(w http.ResponseWriter, r *http.Request) (string, []string, string, string, bool, error) {
	u := r.URL.Path[6:]
	if len(u) == 0 {
		w.Header().Add("Location", "../")
		w.WriteHeader(http.StatusMovedPermanently)
	}
	if match, _ := regexp.MatchString("^[0-9a-f]{32}/.*", u); !match {
		writeResponse(r, w, "Invalid Share Link", "Invalid format for share code.", "")
		return "", []string{}, "", "", false, errors.New("")
	}

	h := u[:32]
	s := queryAccessByShare(h)
	if len(s) == 0 {
		writeResponse(r, w, "Not Found", "Public share code not found.", "")
		return "", []string{}, "", "", false, errors.New("")
	}

	return u[32:], s, h, "", false, nil
}

func handleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "hash", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	aph := r.PostForm.Get("path")
	// //
	queryDoUpdate("shares", "path", aph, "hash", ahs)
	writeAPIResponse(r, w, true, "Successfully updated share path.")
}

func handleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "hash", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	//
	database.QueryPrepared(true, "delete from shares where hash = ?", ahs)
	writeAPIResponse(r, w, true, "Successfully deleted share link.")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if errr != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	writeResponse(r, w, "Success", "Successfully logged out.", "")
}
