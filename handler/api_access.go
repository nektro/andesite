package handler

import (
	"net/http"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"

	"github.com/nektro/andesite/pkg/iutil"
)

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
