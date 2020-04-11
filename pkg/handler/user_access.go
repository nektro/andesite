package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
)

// handler for http://andesite/api/access/create
func HandleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
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
	db.DB.Build().Ins("access", aid, uS, apt).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s.", u.Name+"@"+u.Provider))
}

// handler for http://andesite/api/access/update
func HandleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
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

// handler for http://andesite/api/access/delete
func HandleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	//
	db.DB.Build().Del("access").Wh("id", idS).Exe()
	iutil.WriteAPIResponse(r, w, true, "Removed access "+idS+".")
}
