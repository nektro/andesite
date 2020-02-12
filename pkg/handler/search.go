package handler

import (
	"net/http"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/fsdb"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
)

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
	q, err := hGrabQueryString(r, w, "q")
	if err != nil {
		iutil.WriteJSON(w, map[string]interface{}{
			"response": "bad",
			"message":  "'q' parameter is required",
		})
		return
	}
	fa1 := fsdb.NewFiles(db.FS.QueryPrepared(false, "select * from files where path like '%'||?||'%' limit 25", q))
	ua := iutil.QueryAccess(user)
	fa2 := []itypes.File{}
	//
	for _, item := range fa1 {
		if _, ok := idata.DataPathsPub[item.Root]; ok {
			fa2 = append(fa2, item)
			continue
		}
		if _, ok := idata.DataPathsPrv[item.Root]; ok {
			for _, jtem := range ua {
				if strings.HasPrefix(item.Path, jtem) {
					fa2 = append(fa2, item)
					continue
				}
			}
		}
	}
	iutil.WriteJSON(w, map[string]interface{}{
		"response": "good",
		"count":    len(fa2),
		"results":  fa2,
	})
}
