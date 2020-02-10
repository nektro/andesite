package handler

import (
	"fmt"
	"net/http"
	"strconv"

	etc "github.com/nektro/go.etc"

	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
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

// handler for http://andesite/test
func HandleTest(w http.ResponseWriter, r *http.Request) {
	// sessions test and debug info
	// increment number every refresh
	sess := etc.GetSession(r)
	i := sess.Values["int"]
	if i == nil {
		i = 0
	}
	j := i.(int)
	sess.Values["int"] = j + 1
	sess.Save(r, w)
	fmt.Fprintln(w, strconv.Itoa(j))

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "~~ Host ~~")
	fmt.Fprintln(w, FullHost(r))
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	iutil.WriteLinkResponse(r, w, "Success", "Successfully logged out.", "Back Home", "./../")
}

func HandleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	pk := iutil.GenerateNewUserPasskey(user.Snowflake)
	etc.Database.Build().Up("users", "passkey", pk).Wh("snowflake", user.Snowflake).Exe()
	iutil.WriteLinkResponse(r, w, "Passkey Updated", "It is now: "+pk, "Return", "./files/")
}
