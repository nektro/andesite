package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/iutil"

	. "github.com/nektro/go-util/alias"
)

// handler for http://andesite/api/access/create
func HandleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	_, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	apt := r.PostForm.Get("path")
	//
	db.CreateUserAccess(u, apt)
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s.", u.Name+"@"+u.Provider))
}

// handler for http://andesite/api/access/update
func HandleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	_, id, err := hGrabID(r, w)
	if err != nil {
		return
	}
	ua, ok := db.UserAccess{}.ByID(id)
	if !ok {
		return
	}
	_, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	apt := r.PostForm.Get("path")
	//
	ua.SetUser(u)
	ua.SetPath(apt)
	iutil.WriteAPIResponse(r, w, true, "Updated access for "+u.Name+"@"+u.Provider+".")
}

// handler for http://andesite/api/access/delete
func HandleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	idS, id, err := hGrabID(r, w)
	if err != nil {
		return
	}
	ua, ok := db.UserAccess{}.ByID(id)
	if !ok {
		return
	}
	//
	ua.Delete()
	iutil.WriteAPIResponse(r, w, true, "Removed access "+idS+".")
}
