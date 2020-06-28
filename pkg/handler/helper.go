package handler

import (
	"net/http"
	"strconv"

	"github.com/nektro/andesite/pkg/db"

	. "github.com/nektro/go-util/alias"
)

func hGrabUser(r *http.Request, w http.ResponseWriter) (string, *db.User, error) {
	if !ContainsAll(r.PostForm, "user") {
		WriteAPIResponse(r, w, false, "Missing POST value: user")
		return "", nil, E("")
	}
	a := r.PostForm.Get("user")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		WriteAPIResponse(r, w, false, "User parameter must be an integer")
		return a, nil, E("")
	}
	u, ok := db.User{}.ByID(n)
	if !ok {
		WriteLinkResponse(r, w, "Unable to find User", "Invalid user ID.", "Return", "./../../admin")
		return a, nil, E("")
	}
	return a, u, nil
}
