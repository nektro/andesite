package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/iutil"

	. "github.com/nektro/go-util/alias"
)

func HandleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	fpath := r.PostForm.Get("path")
	//
	sh := db.CreateShare(fpath)
	iutil.WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", sh.Hash, fpath))
}

func HandleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	_, id, err := hGrabID(r, w)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	aph := r.PostForm.Get("path")
	sh, ok := db.Share{}.ByID(id)
	if !ok {
		return
	}
	//
	sh.SetPath(aph)
	iutil.WriteAPIResponse(r, w, true, "Successfully updated share path.")
}

func HandleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	_, id, err := hGrabID(r, w)
	if err != nil {
		return
	}
	sh, ok := db.Share{}.ByID(id)
	if !ok {
		return
	}
	//
	sh.Delete()
	iutil.WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}
