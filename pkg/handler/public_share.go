package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"

	"github.com/nektro/go.etc/htp"

	. "github.com/nektro/go-util/alias"
)

func HandleShareCreate(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	fpath := c.GetFormString("path")
	sh := db.CreateShare(fpath)
	WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", sh.Hash, fpath))
}

func HandleShareUpdate(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	_, id := c.GetFormInt("id")
	aph := c.GetFormString("path")
	sh, ok := db.Share{}.ByID(id)
	if !ok {
		return
	}
	sh.SetPath(aph)
	WriteAPIResponse(r, w, true, "Successfully updated share path.")
}

func HandleShareDelete(w http.ResponseWriter, r *http.Request) {
	c := htp.GetController(r)
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	_, id := c.GetFormInt("id")
	sh, ok := db.Share{}.ByID(id)
	if !ok {
		return
	}
	sh.Delete()
	WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}
