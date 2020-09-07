package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"
)

func Init() {
	etc.HtpErrCb = func(r *http.Request, w http.ResponseWriter, good bool, code int, msg string) {
		WriteAPIResponse(r, w, good, msg)
	}
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
	fmt.Fprintln(w, util.FullHost(r))
}

// handler for http://andesite/logout
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, err := ApiBootstrap(r, w, []string{http.MethodGet}, true, false, true)
	if err != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	WriteLinkResponse(r, w, "Success", "Successfully logged out.", "Back Home", "./")
}

func HandleRegenPasskey(w http.ResponseWriter, r *http.Request) {
	_, user, err := ApiBootstrap(r, w, []string{http.MethodGet}, true, false, true)
	if err != nil {
		return
	}
	user.ResetPasskey()
	WriteLinkResponse(r, w, "Passkey Updated", "It is now: "+user.PassKey, "Return", "./files/")
}
