package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	etc "github.com/nektro/go.etc"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

func helperIsLoggedIn(r *http.Request) bool {
	sess := etc.GetSession(r)
	_, ok := sess.Values["user"]
	return ok
}

func helperOA2SaveInfo(w http.ResponseWriter, r *http.Request, provider string, id string, name string, resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Values[provider+"_access_token"] = resp["access_token"]
	sess.Values[provider+"_expires_in"] = resp["expires_in"]
	sess.Values[provider+"_refresh_token"] = resp["refresh_token"]
	sess.Save(r, w)
	queryAssertUserName(id, name)
	Log("[user-login]", provider, id, name)
}

// handler for http://andesite/test
func handleTest(w http.ResponseWriter, r *http.Request) {
	// sessions test and debug info
	// increment number every refresh
	sess := etc.GetSession(r)
	i := sess.Values["int"]
	if i == nil {
		i = 0
	}
	j := i.(int)
	sess.Values["int"] = j + 1
	sess.Save(r, w)
	fmt.Fprintln(w, strconv.Itoa(j))

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "~~ Host ~~")
	fmt.Fprintln(w, FullHost(r))
}

func handleDirectoryListing(getAccess func(http.ResponseWriter, *http.Request) (string, string, []string, string, string, bool, map[string]interface{}, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fileRoot, qpath, uAccess, uID, uName, isAdmin, extras, err := getAccess(w, r)

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
		stat, err := os.Stat(fileRoot + qpath)
		if os.IsNotExist(err) {
			// 404
			writeUserDenied(r, w, true, false)
			return
		}

		// server file/folder
		if stat.IsDir() {
			w.Header().Add("Content-Type", "text/html")

			// get list of all files
			files, _ := ioutil.ReadDir(fileRoot + qpath)

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

			etc.WriteHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
				"user":      uID,
				"path":      qpath,
				"files":     data,
				"admin":     isAdmin,
				"base":      config.HTTPBase,
				"name":      oauth2Provider.IDP.NamePrefix + uName,
				"search_on": config.SearchOn,
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
				writeUserDenied(r, w, true, false)
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
func handleFileListing(w http.ResponseWriter, r *http.Request) (string, string, []string, string, string, bool, map[string]interface{}, error) {
	_, user, err := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if err != nil {
		return "", "", []string{}, "", "", false, map[string]interface{}{}, err
	}

	// get path
	// remove /files
	qpath := string(r.URL.Path[6:])

	userUser, _ := queryUserBySnowflake(user.Snowflake)
	userAccess := queryAccess(user)

	if oauth2Provider.IDP.ID == "discord" {
		dra := queryAllDiscordRoleAccess()
		var p fastjson.Parser

		rurl := F("%s/guilds/%s/members/%s", discordAPI, config.Discord.Extra1, user.Snowflake)
		req, _ := http.NewRequest(http.MethodGet, rurl, strings.NewReader(""))
		req.Header.Set("User-Agent", "nektro/andesite")
		// req.Header.Set("Authorization", "Bearer "+sess.Values[accessToken].(string))
		req.Header.Set("Authorization", "Bot "+config.Discord.Extra2)
		bys := doHttpRequest(req)
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

	return config.Root, qpath, userAccess, user.Snowflake, user.Name, userUser.Admin, map[string]interface{}{
		"user": user,
	}, nil
}

// handler for http://andesite/public/*
func handlePublicListing(w http.ResponseWriter, r *http.Request) (string, string, []string, string, string, bool, map[string]interface{}, error) {
	// remove /public
	qpath := string(r.URL.Path[7:])
	qaccess := []string{}

	if len(config.Public) == 0 {
		return config.Public, qpath, qaccess, "", "new member!", false, map[string]interface{}{}, nil
	}
	return config.Public, qpath, []string{"/"}, "", "new member", false, map[string]interface{}{}, nil
}

// handler for http://andesite/admin
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := apiBootstrapRequireLogin(r, w, http.MethodGet, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"user":           user.Snowflake,
		"accesses":       queryAllAccess(),
		"base":           config.HTTPBase,
		"name":           oauth2Provider.IDP.NamePrefix + user.Name,
		"shares":         queryAllShares(),
		"auth":           oauth2Provider.IDP.ID,
		"discord_shares": queryAllDiscordRoleAccess(),
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
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
	etc.Database.Query(true, F("delete from access where id = '%d'", iid))
	writeAPIResponse(r, w, true, F("Removed access from %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
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
	writeAPIResponse(r, w, true, F("Updated access for %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "snowflake", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("access")
	asn := r.PostForm.Get("snowflake")
	apt := r.PostForm.Get("path")
	//
	u, ok := queryUserBySnowflake(asn)
	aud := -1
	if ok {
		aud = u.ID
	} else {
		aud = etc.Database.QueryNextID("users")
		queryDoAddUser(aud, asn, false, "")
	}
	//
	etc.Database.QueryPrepared(true, "insert into access values (?, ?, ?)", aid, aud, apt)
	writeAPIResponse(r, w, true, F("Created access for %s.", asn))
}

func handleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("shares")
	ahs1 := md5.Sum([]byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(int64(aid), 10), GetIsoDateTime())))
	ahs2 := hex.EncodeToString(ahs1[:])
	fpath := r.PostForm.Get("path")
	//
	etc.Database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ahs2, fpath)
	writeAPIResponse(r, w, true, F("Created share with code %s for folder %s.", ahs2, fpath))
}

func handleShareListing(w http.ResponseWriter, r *http.Request) (string, string, []string, string, string, bool, map[string]interface{}, error) {
	u := r.URL.Path[6:]
	if len(u) == 0 {
		w.Header().Add("Location", "../")
		w.WriteHeader(http.StatusFound)
	}
	if match, _ := regexp.MatchString("^[0-9a-f]{32}/.*", u); !match {
		writeResponse(r, w, "Invalid Share Link", "Invalid format for share code.", "")
		return "", "", []string{}, "", "", false, map[string]interface{}{}, errors.New("")
	}

	h := u[:32]
	s := queryAccessByShare(h)
	if len(s) == 0 {
		writeResponse(r, w, "Not Found", "Public share code not found.", "")
		return "", "", []string{}, "", "", false, map[string]interface{}{}, errors.New("")
	}

	return config.Root, u[32:], s, h, "", false, map[string]interface{}{}, nil
}

func handleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
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
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
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
	etc.Database.QueryPrepared(true, "delete from shares where hash = ?", ahs)
	writeAPIResponse(r, w, true, "Successfully deleted share link.")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, err := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if err != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	writeResponse(r, w, "Success", "Successfully logged out.", "")
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	_, user, err := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if err != nil {
		return
	}
	//
	etc.WriteHandlebarsFile(r, w, "/search.hbs", map[string]interface{}{
		"user": user.Snowflake,
		"base": config.HTTPBase,
		"name": oauth2Provider.IDP.NamePrefix + user.Name,
	})
}

func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	_, user, err := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"response": "bad",
			"message":  err.Error(),
		})
		return
	}
	p := r.URL.Query()["q"]
	if len(p) == 0 || len(p[0]) == 0 {
		writeJSON(w, map[string]interface{}{
			"response": "bad",
			"message":  "'q' parameter is required",
		})
		return
	}
	//
	v0 := p[0]
	v1 := strings.Replace(v0, "!", "!!", -1)
	v2 := strings.Replace(v1, "%", "!%", -1)
	v3 := strings.Replace(v2, "_", "!_", -1)
	v4 := strings.Replace(v3, "[", "![", -1)
	a := []WatchedFile{}
	ua := queryAccess(user)
	q := etc.Database.QueryPrepared(false, "select * from files where path like ? escape '!'", "%"+v4+"%")
	for q.Next() {
		wf := scanFile(q)
		wf.URL = config.HTTPBase + "files" + wf.Path
		//
		if strings.Contains(wf.Path, "/.") {
			continue
		}
		for _, item := range ua {
			if strings.HasPrefix(wf.Path, item) {
				a = append(a, wf)
				break
			}
		}
		if len(a) == 25 {
			break
		}
	}
	q.Close()
	writeJSON(w, map[string]interface{}{
		"response": "good",
		"count":    len(a),
		"results":  a,
	})
}

func handleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "RoleID", "Path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("shares_discord_role")
	// ags := r.PostForm.Get("GuildID")
	ags := config.Discord.Extra1
	agr := r.PostForm.Get("RoleID")
	apt := r.PostForm.Get("Path")
	//
	etc.Database.QueryPrepared(true, "insert into shares_discord_role values (?, ?, ?, ?)", aid, ags, agr, apt)
	writeAPIResponse(r, w, true, F("Created access for %s / %s.", ags, agr))
}

func handleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "ID", "RoleID", "Path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	qid := r.PostForm.Get("ID")
	// qgs := r.PostForm.Get("GuildID")
	qgs := config.Discord.Extra1
	qgr := r.PostForm.Get("RoleID")
	qpt := r.PostForm.Get("Path")
	//
	rn := fetchDiscordRole(config.Discord.Extra1, qgr).Name
	//
	queryDoUpdate("shares_discord_role", "guild_snowflake", qgs, "id", qid)
	queryDoUpdate("shares_discord_role", "role_snowflake", qgr, "id", qid)
	queryDoUpdate("shares_discord_role", "path", qpt, "id", qid)
	queryDoUpdate("shares_discord_role", "guild_name", "_", "id", qid)
	queryDoUpdate("shares_discord_role", "role_name", rn, "id", qid)
	writeAPIResponse(r, w, true, "Successfully updated share path.")
}

func handleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if err != nil {
		return
	}
	if !containsAll(r.PostForm, "ID") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	qID := r.PostForm.Get("ID")
	etc.Database.QueryPrepared(true, "delete from shares_discord_role where id = ?", qID)
	writeAPIResponse(r, w, true, "Successfully deleted share link.")
}

//
func handleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if err != nil {
		return
	}
	pk := generateNewUserPasskey(user.Snowflake)
	etc.Database.QueryDoUpdate("users", "passkey", pk, "snowflake", user.Snowflake)
	writeLinkResponse(r, w, "Passkey Updated", "It is now: "+pk, "Return", "./files/")
}
