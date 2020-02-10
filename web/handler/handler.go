package handler

import (
	"fmt"
	"net/http"
	"strconv"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/util"
)

func hGrabID(r *http.Request, w http.ResponseWriter) (string, int64, error) {
	if !util.ContainsAll(r.PostForm, "id") {
		util.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", -1, E("")
	}
	a := r.PostForm.Get("id")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		util.WriteAPIResponse(r, w, false, "ID parameter must be an integer")
		return a, -1, E("")
	}
	return a, n, nil
}

func hGrabUser(r *http.Request, w http.ResponseWriter) (string, *db.UserRow, error) {
	if !util.ContainsAll(r.PostForm, "user") {
		util.WriteAPIResponse(r, w, false, "Missing POST values")
		return "", nil, E("")
	}
	a := r.PostForm.Get("user")
	n, err := strconv.ParseInt(a, 10, 64)
	if err != nil {
		util.WriteAPIResponse(r, w, false, "User parameter must be an integer")
		return a, nil, E("")
	}
	u, ok := db.QueryUserByID(n)
	if !ok {
		util.WriteLinkResponse(r, w, "Unable to find User", "Invalid user ID.", "Return", "./../../admin")
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
	sess, _, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	util.WriteLinkResponse(r, w, "Success", "Successfully logged out.", "Back Home", "./../")
}

func HandleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, false)
	if err != nil {
		return
	}
	pk := util.GenerateNewUserPasskey(user.Snowflake)
	etc.Database.Build().Up("users", "passkey", pk).Wh("snowflake", user.Snowflake).Exe()
	util.WriteLinkResponse(r, w, "Passkey Updated", "It is now: "+pk, "Return", "./files/")
}
