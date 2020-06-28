package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"

	"github.com/nektro/go.etc/htp"

	. "github.com/nektro/go-util/alias"
)

// handler for http://andesite/api/access/create
func HandleAccessCreate(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	_, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	//
	apt := c.GetFormString("path")
	db.CreateUserAccess(u, apt)
	WriteAPIResponse(r, w, true, F("Created access for %s.", u.Name+"@"+u.Provider))
}

// handler for http://andesite/api/access/update
func HandleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	_, id := c.GetFormInt("id")
	ua, ok := db.UserAccess{}.ByID(id)
	if !ok {
		return
	}
	_, u, err := hGrabUser(r, w)
	if err != nil {
		return
	}
	//
	apt := c.GetFormString("path")
	ua.SetUser(u)
	ua.SetPath(apt)
	WriteAPIResponse(r, w, true, "Updated access for "+u.FullName()+".")
}

// handler for http://andesite/api/access/delete
func HandleAccessDelete(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	idS, id := c.GetFormInt("id")
	ua, ok := db.UserAccess{}.ByID(id)
	if !ok {
		return
	}
	//
	ua.Delete()
	WriteAPIResponse(r, w, true, "Removed access "+idS+".")
}
