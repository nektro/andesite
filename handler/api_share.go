package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"
)

func HandleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	aid := etc.Database.QueryNextID("shares")
	ash := Hash("MD5", []byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(aid, 10), GetIsoDateTime())))[:12]
	if !iutil.ContainsAll(r.PostForm, "path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	fpath := r.PostForm.Get("path")
	if !strings.HasSuffix(fpath, "/") {
		fpath += "/"
	}
	//
	etc.Database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ash, fpath)
	iutil.WriteAPIResponse(r, w, true, F("Created share with code %s for folder %s.", ash, fpath))
}

func HandleShareListing(w http.ResponseWriter, r *http.Request) (string, string, []string, *itypes.UserRow, map[string]interface{}, error) {
	u := strings.Split(r.URL.Path, "/")
	if len(u) <= 3 {
		w.Header().Add("Location", "../")
		w.WriteHeader(http.StatusFound)
	}
	h := u[2]
	s := iutil.QueryAccessByShare(h)
	if len(s) == 0 {
		iutil.WriteResponse(r, w, "Not Found", "", "")
		return "", "", nil, nil, nil, errors.New("")
	}
	sp := strings.Split(s, "/")
	dp, ok := idata.DataPaths[sp[1]]
	if !ok {
		iutil.WriteResponse(r, w, "Not Found", "", "")
		return "", "", nil, nil, nil, errors.New("")
	}
	return dp + "/" + strings.Join(sp[2:], "/"), "/" + strings.Join(u[3:], "/"), []string{"/"}, &itypes.UserRow{ID: -1, Name: "Guest", Provider: r.Host}, nil, nil
}

func HandleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
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
	// //
	etc.Database.Build().Up("shares", "path", aph).Wh("id", idS).Exe()
	iutil.WriteAPIResponse(r, w, true, "Successfully updated share path.")
}

func HandleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
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
	etc.Database.QueryPrepared(true, "delete from shares where id = ?", idS)
	iutil.WriteAPIResponse(r, w, true, "Successfully deleted share link.")
}
