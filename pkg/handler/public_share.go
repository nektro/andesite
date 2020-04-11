package handler

import (
	"net/http"
	"strconv"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/iutil"

	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
)

func HandleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	aid := etc.Database.QueryNextID("shares")
	ash := util.Hash("MD5", []byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(aid, 10), T())))[:12]
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	fpath := r.PostForm.Get("path")
	//
	db.DB.Build().Ins("shares", aid, ash, fpath).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", ash, fpath))
}

func HandleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
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
	//
	etc.Database.Build().Up("shares", "path", aph).Wh("id", idS).Exe()
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
	idS, _, err := hGrabID(r, w)
	if err != nil {
		return
	}
	//
	db.DB.Build().Del("shares").Wh("id", idS).Exe()
	iutil.WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}
