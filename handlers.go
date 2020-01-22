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

	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
	"github.com/valyala/fastjson"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

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

func handleDirectoryListing(getAccess func(http.ResponseWriter, *http.Request) (string, string, []string, *itypes.UserRow, map[string]interface{}, error)) func(http.ResponseWriter, *http.Request) {
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
			iutil.WriteUserDenied(r, w, true, false)
			return
		}

		// valid path check
		stat, err := os.Stat(fileRoot + qpath)
		if os.IsNotExist(err) {
			// 404
			iutil.WriteUserDenied(r, w, true, false)
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
			files = iutil.Filter(files, func(x os.FileInfo) bool {
				return !strings.HasPrefix(x.Name(), ".")
			})

			// amount of files in the directory
			l1 := len(files)

			// access check
			files = iutil.Filter(files, func(x os.FileInfo) bool {
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
				iutil.WriteUserDenied(r, w, true, false)
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
					"size": ByteCountIEC(files[i].Size()),
					"mod":  files[i].ModTime().UTC().String()[:19],
					"ext":  ext[1:],
				}
				gi++
			}

			etc.WriteHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
				"provider":  user.Provider,
				"user":      user.Snowflake,
				"path":      qpath,
				"files":     data,
				"admin":     user.Admin,
				"base":      idata.Config.HTTPBase,
				"name":      oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
				"search_on": idata.Config.SearchOn,
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
				iutil.WriteUserDenied(r, w, true, false)
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
func handleFileListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *itypes.UserRow, map[string]interface{}, error) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet, http.MethodHead}, false)
	if err != nil {
		return "", "", nil, nil, nil, err
	}

	// get path
	// remove /files
	qpath := string(r.URL.Path[6:])

	userAccess := iutil.QueryAccess(user)

	if user.Provider == oauth2.ProviderIDMap["discord"].ID && idata.Config.GetDiscordClient().Extra1 != "" && idata.Config.GetDiscordClient().Extra2 != "" {
		dra := iutil.QueryAllDiscordRoleAccess()
		var p fastjson.Parser

		rurl := F("%s/guilds/%s/members/%s", idata.DiscordAPI, idata.Config.GetDiscordClient().Extra1, user.Snowflake)
		req, _ := http.NewRequest(http.MethodGet, rurl, strings.NewReader(""))
		req.Header.Set("User-Agent", "nektro/andesite")
		req.Header.Set("Authorization", "Bot "+idata.Config.GetDiscordClient().Extra2)
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

	return idata.Config.Root, qpath, userAccess, user, map[string]interface{}{
		"user": user,
	}, nil
}

// handler for http://andesite/public/*
func handlePublicListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *itypes.UserRow, map[string]interface{}, error) {
	// remove /public
	qpath := string(r.URL.Path[7:])
	qaccess := []string{}

	if len(idata.Config.Public) > 0 {
		qaccess = append(qaccess, "/")
	}
	return idata.Config.Public, qpath, qaccess, &itypes.UserRow{-1, "", "", false, "Guest!", "", "", ""}, map[string]interface{}{}, nil
}

// handler for http://andesite/admin
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"user": user.Snowflake,
		"base": idata.Config.HTTPBase,
		"name": oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth": oauth2.ProviderIDMap[user.Provider].ID,
		"accesses":       iutil.QueryAllAccess(),
		"shares":         iutil.QueryAllShares(),
		"discord_shares": iutil.QueryAllDiscordRoleAccess(),
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "id", "snowflake") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := r.PostForm.Get("id")
	iid, err := strconv.ParseInt(aid, 10, 32)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	etc.Database.QueryPrepared(true, "delete from access where id = ?", iid)
	iutil.WriteAPIResponse(r, w, true, F("Removed access from %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	aid := r.PostForm.Get("id")
	_, err = strconv.ParseInt(string(aid), 10, 64)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "snowflake", "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	etc.Database.Build().Up("access", "path", r.PostForm.Get("path")).Wh("id", aid).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Updated access for %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "snowflake", "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("access")
	asn := r.PostForm.Get("snowflake")
	apt := r.PostForm.Get("path")
	//
	u, ok := iutil.QueryUserBySnowflake(asn)
	aud := int64(-1)
	if ok {
		aud = u.ID
	} else {
		aud = etc.Database.QueryNextID("users")
		iutil.QueryDoAddUser(aud, oauth2.ProviderIDMap[user.Provider].ID, asn, false, "")
	}
	//
	etc.Database.QueryPrepared(true, "insert into access values (?, ?, ?)", aid, aud, apt)
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s.", asn))
}

func handleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("shares")
	ahs1 := md5.Sum([]byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(int64(aid), 10), GetIsoDateTime())))
	ahs2 := hex.EncodeToString(ahs1[:])
	fpath := r.PostForm.Get("path")
	//
	etc.Database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ahs2, fpath)
	iutil.WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", ahs2, fpath))
}

func handleShareListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *itypes.UserRow, map[string]interface{}, error) {
	u := r.URL.Path[6:]
	if len(u) == 0 {
		w.Header().Add("Location", "../")
		w.WriteHeader(http.StatusFound)
	}
	if match, _ := regexp.MatchString("^[0-9a-f]{32}/.*", u); !match {
		iutil.WriteResponse(r, w, "Invalid Share Link", "Invalid format for share code.", "")
		return "", "", nil, nil, nil, errors.New("")
	}

	h := u[:32]
	s := iutil.QueryAccessByShare(h)
	if len(s) == 0 {
		iutil.WriteResponse(r, w, "Not Found", "Public share code not found.", "")
		return "", "", nil, nil, nil, errors.New("")
	}

	return idata.Config.Root, u[32:], s, &itypes.UserRow{-1, "", h, false, "", "", "", ""}, nil, nil
}

func handleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "id", "hash", "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	aph := r.PostForm.Get("path")
	// //
	etc.Database.Build().Up("shares", "path", aph).Wh("hash", ahs).Exe()
	iutil.WriteAPIResponse(r, w, true, "Successfully updated share path.")
}

func handleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "id", "hash", "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	//
	etc.Database.QueryPrepared(true, "delete from shares where hash = ?", ahs)
	iutil.WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	iutil.WriteLinkResponse(r, w, "Success", "Successfully logged out.", "Back Home", "./../")
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	//
	etc.WriteHandlebarsFile(r, w, "/search.hbs", map[string]interface{}{
		"user": user.Snowflake,
		"base": idata.Config.HTTPBase,
		"name": oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
	})
}

func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		iutil.WriteJSON(w, map[string]interface{}{
			"response": "bad",
			"message":  err.Error(),
		})
		return
	}
	p := r.URL.Query()["q"]
	if len(p) == 0 || len(p[0]) == 0 {
		iutil.WriteJSON(w, map[string]interface{}{
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
	ua := iutil.QueryAccess(user)
	q := etc.Database.QueryPrepared(false, "select * from files where path like ? escape '!'", "%"+v4+"%")
	for q.Next() {
		wf := scanFile(q)
		wf.URL = idata.Config.HTTPBase + "files" + wf.Path
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
	iutil.WriteJSON(w, map[string]interface{}{
		"response": "good",
		"count":    len(a),
		"results":  a,
	})
}

func handleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "RoleID", "Path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("shares_discord_role")
	// ags := r.PostForm.Get("GuildID")
	ags := idata.Config.GetDiscordClient().Extra1
	agr := r.PostForm.Get("RoleID")
	apt := r.PostForm.Get("Path")
	//
	gn := iutil.FetchDiscordGuild(idata.Config.GetDiscordClient().Extra1).Name
	rn := iutil.FetchDiscordRole(idata.Config.GetDiscordClient().Extra1, agr).Name
	//
	etc.Database.QueryPrepared(true, "insert into shares_discord_role values (?, ?, ?, ?, ?, ?)", aid, ags, agr, apt, gn, rn)
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s / %s to %s.", gn, rn, apt))
}

func handleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "ID", "RoleID", "Path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	qid := r.PostForm.Get("ID")
	// qgs := r.PostForm.Get("GuildID")
	qgs := idata.Config.GetDiscordClient().Extra1
	qgr := r.PostForm.Get("RoleID")
	qpt := r.PostForm.Get("Path")
	//
	gn := iutil.FetchDiscordGuild(idata.Config.GetDiscordClient().Extra1).Name
	rn := iutil.FetchDiscordRole(idata.Config.GetDiscordClient().Extra1, qgr).Name
	//
	etc.Database.Build().Up("shares_discord_role", "guild_snowflake", qgs).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_snowflake", qgr).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "path", qpt).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "guild_name", gn).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_name", rn).Wh("id", qid).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Successfully updated share path for %s / %s to %s.", gn, rn, qpt))
}

func handleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "ID") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	qID, err := strconv.ParseInt(r.PostForm.Get("ID"), 10, 64)
	if err != nil {
		return
	}
	dra := iutil.QueryDiscordRoleAccess(qID)
	if dra == nil {
		return
	}
	etc.Database.QueryPrepared(true, "delete from shares_discord_role where id = ?", qID)
	iutil.WriteAPIResponse(r, w, true, F("Successfully deleted access for %s / %s to %s.", dra.GuildName, dra.RoleName, dra.Path))
}

//
func handleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	pk := iutil.GenerateNewUserPasskey(user.Snowflake)
	etc.Database.Build().Up("users", "passkey", pk).Wh("snowflake", user.Snowflake).Exe()
	iutil.WriteLinkResponse(r, w, "Passkey Updated", "It is now: "+pk, "Return", "./files/")
}

//
func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/users.hbs", map[string]interface{}{
		"user":  user.Snowflake,
		"base":  idata.Config.HTTPBase,
		"name":  oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":  oauth2.ProviderIDMap[user.Provider].ID,
		"users": iutil.QueryAllUsers(),
	})
}
