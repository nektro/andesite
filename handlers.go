package main

import (
	"net/http"
	"strings"

	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
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
	util.Log("[user-login]", provider, id, name)
}

//
//

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
