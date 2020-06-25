package handler

import (
	"net/http"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/fsdb"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
)

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrap(r, w, []string{http.MethodGet}, false, false, true)
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
	_, user, err := iutil.ApiBootstrap(r, w, []string{http.MethodGet}, false, false, false)
	if err != nil {
		iutil.WriteJSON(w, map[string]interface{}{
			"response": "bad",
			"message":  err.Error(),
		})
		return
	}
	q := db.FS.Build().Se("*").Fr("files")
	{
		qq := r.Form.Get("q")
		if len(qq) > 0 {
			q.WR("path", "like", "'%'||?||'%'", true, qq)
		}
	}
	for _, item := range []string{"md5", "sha1", "sha256", "sha512", "sha3", "blake2b"} {
		qh := r.Form.Get(item)
		if len(qh) > 0 {
			q.Wh("hash_"+item, qh)
		}
	}
	fa1 := fsdb.NewFiles(q.Lm(25).Exe())
	ua := db.QueryAccess(user)
	fa2 := []*db.File{}
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
