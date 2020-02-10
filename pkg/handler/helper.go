package handler

import (
	"net/http"
	"strconv"

	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

	. "github.com/nektro/go-util/alias"
)

func hGrabID(r *http.Request, w http.ResponseWriter) (string, int64, error) {
	if !iutil.ContainsAll(r.PostForm, "id") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", -1, E("")
	}
	a := r.PostForm.Get("id")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "ID parameter must be an integer")
		return a, -1, E("")
	}
	return a, n, nil
}

func hGrabUser(r *http.Request, w http.ResponseWriter) (string, *itypes.UserRow, error) {
	if !iutil.ContainsAll(r.PostForm, "user") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", nil, E("")
	}
	a := r.PostForm.Get("user")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		iutil.WriteAPIResponse(r, w, false, "User parameter must be an integer")
		return a, nil, E("")
	}
	u, ok := iutil.QueryUserByID(n)
	if !ok {
		iutil.WriteLinkResponse(r, w, "Unable to find User", "Invalid user ID.", "Return", "./../../admin")
		return a, nil, E("")
	}
	return a, u, nil
}
