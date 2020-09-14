package handler

import (
	"net/http"

	etc "github.com/nektro/go.etc"
)

func Init() {
	etc.HtpErrCb = func(r *http.Request, w http.ResponseWriter, good bool, code int, msg string) {
		WriteAPIResponse(r, w, good, msg)
	}
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
