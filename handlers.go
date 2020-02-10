package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

//
//

func hGrabID(r *http.Request, w http.ResponseWriter) (string, int64, error) {
	if !iutil.ContainsAll(r.PostForm, "id") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", -1, E("")
	}
	a := r.PostForm.Get("id")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "ID parameter must be an integer")
		return a, -1, E("")
	}
	return a, n, nil
}

func hGrabUser(r *http.Request, w http.ResponseWriter) (string, *itypes.UserRow, error) {
	if !iutil.ContainsAll(r.PostForm, "user") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", nil, E("")
	}
	a := r.PostForm.Get("user")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "User parameter must be an integer")
		return a, nil, E("")
	}
	u, ok := iutil.QueryUserByID(n)
	if !ok {
		iutil.WriteLinkResponse(r, w, "Unable to find User", "Invalid user ID.", "Return", "./../../admin")
		return a, nil, E("")
	}
	return a, u, nil
}

//
//

// handler for http://andesite/admin
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	dc := idata.Config.GetDiscordClient()
	etc.WriteHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"version":               idata.Version,
		"user":                  user,
		"base":                  idata.Config.HTTPBase,
		"name":                  oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":                  oauth2.ProviderIDMap[user.Provider].ID,
		"discord_role_share_on": len(dc.Extra1) > 0 && len(dc.Extra2) > 0,
		"users":                 iutil.QueryAllUsers(),
		"accesses":              iutil.QueryAllAccess(),
		"shares":                iutil.QueryAllShares(),
		"discord_shares":        iutil.QueryAllDiscordRoleAccess(),
	})
}

// handler for http://andesite/api/access/delete
func HandleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	//
	etc.Database.QueryPrepared(true, "delete from access where id = ?", idS)
	iutil.WriteAPIResponse(r, w, true, "Removed access "+idS+".")
}

// handler for http://andesite/api/access/update
func HandleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	uS, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	apt := r.PostForm.Get("path")
	//
	etc.Database.Build().Up("access", "user", uS).Wh("id", idS).Exe()
	etc.Database.Build().Up("access", "path", apt).Wh("id", idS).Exe()
	iutil.WriteAPIResponse(r, w, true, "Updated access for "+u.Name+"@"+u.Provider+".")
}

// handler for http://andesite/api/access/create
func HandleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	aid := etc.Database.QueryNextID("access")
	uS, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	apt := r.PostForm.Get("path")
	//
	etc.Database.QueryPrepared(true, "insert into access values (?, ?, ?)", aid, uS, apt)
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s.", u.Name+"@"+u.Provider))
}

func HandleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	aid := etc.Database.QueryNextID("shares")
	ash := Hash("MD5", []byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(aid, 10), GetIsoDateTime())))[:12]
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	fpath := r.PostForm.Get("path")
	if !strings.HasSuffix(fpath, "/") {
		fpath += "/"
	}
	//
	etc.Database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ash, fpath)
	iutil.WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", ash, fpath))
}

func HandleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	aph := r.PostForm.Get("path")
	// //
	etc.Database.Build().Up("shares", "path", aph).Wh("id", idS).Exe()
	iutil.WriteAPIResponse(r, w, true, "Successfully updated share path.")
}

func HandleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	//
	etc.Database.QueryPrepared(true, "delete from shares where id = ?", idS)
	iutil.WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	//
	etc.WriteHandlebarsFile(r, w, "/search.hbs", map[string]interface{}{
		"version": idata.Version,
		"user":    user,
		"base":    idata.Config.HTTPBase,
		"name":    oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
	})
}

func HandleSearchAPI(w http.ResponseWriter, r *http.Request) {
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

func HandleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
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
	gn := iutil.FetchDiscordGuild(ags).Name
	rn := iutil.FetchDiscordRole(ags, agr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		iutil.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	etc.Database.QueryPrepared(true, "insert into shares_discord_role values (?, ?, ?, ?, ?, ?)", aid, ags, agr, apt, gn, rn)
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s / %s to %s.", gn, rn, apt))
}

func HandleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
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
	gn := iutil.FetchDiscordGuild(qgs).Name
	rn := iutil.FetchDiscordRole(qgs, qgr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		iutil.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	etc.Database.Build().Up("shares_discord_role", "guild_snowflake", qgs).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_snowflake", qgr).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "path", qpt).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "guild_name", gn).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_name", rn).Wh("id", qid).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Successfully updated share path for %s / %s to %s.", gn, rn, qpt))
}

func HandleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
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
func HandleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	pk := iutil.GenerateNewUserPasskey(user.Snowflake)
	etc.Database.Build().Up("users", "passkey", pk).Wh("snowflake", user.Snowflake).Exe()
	iutil.WriteLinkResponse(r, w, "Passkey Updated", "It is now: "+pk, "Return", "./files/")
}

//
func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/users.hbs", map[string]interface{}{
		"version": idata.Version,
		"user":    user,
		"base":    idata.Config.HTTPBase,
		"name":    oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":    oauth2.ProviderIDMap[user.Provider].ID,
		"users":   iutil.QueryAllUsers(),
	})
}
