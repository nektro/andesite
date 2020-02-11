package db

import (
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	. "github.com/nektro/go-util/alias"
	goUtil "github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"

	"github.com/nektro/andesite/util"
)

func ApiBootstrapRequireLogin(r *http.Request, w http.ResponseWriter, methods []string, requireAdmin bool) (*sessions.Session, *UserRow, error) {
	if !goUtil.Contains(methods, r.Method) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Add("Allow", F("%v", methods))
		util.WriteAPIResponse(r, w, false, "This action requires using HTTP "+F("%v", methods))
		return nil, nil, E("")
	}

	sess := etc.GetSession(r)
	provID := sess.Values["provider"]
	sessID := sess.Values["user"]

	if sessID == nil {
		pk := ""

		if len(pk) == 0 {
			ua := r.Header.Get("user-agent")
			if strings.HasPrefix(ua, "AndesiteUser/") {
				pk = strings.Split(ua, "/")[1]
			}
		}
		if len(pk) == 0 {
			pk = r.Header.Get("x-passkey")
		}
		if len(pk) == 0 {
			util.WriteUserDenied(r, w, true, true)
			return nil, nil, E("not logged in and no passkey found")
		}
		kq := etc.Database.Build().Se("*").Fr("users").Wh("passkey", pk).Exe()
		if !kq.Next() {
			util.WriteUserDenied(r, w, true, true)
			return nil, nil, E("invalid passkey")
		}
		sessID = ScanUser(kq).Snowflake
		kq.Close()
	}

	pS := provID.(string)
	uS := sessID.(string)
	user, ok := QueryUserBySnowflake(pS, uS)

	if !ok {
		util.WriteResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+uS+"@"+pS+")", "")
		return nil, nil, E("")
	}
	if requireAdmin && !user.Admin {
		util.WriteAPIResponse(r, w, false, "This action requires being a site administrator. ("+uS+"@"+pS+")")
		return nil, nil, E("")
	}

	err := r.ParseForm()
	if err != nil {
		util.WriteAPIResponse(r, w, false, "Error parsing form data")
		return nil, nil, E("")
	}

	return sess, user, nil
}

func HelperOA2SaveInfo(w http.ResponseWriter, r *http.Request, provider string, id string, name string, resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["provider"] = provider
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Values[provider+"_access_token"] = resp["access_token"]
	sess.Values[provider+"_expires_in"] = resp["expires_in"]
	sess.Values[provider+"_refresh_token"] = resp["refresh_token"]
	sess.Save(r, w)
	QueryAssertUserName(provider, id, name)
	goUtil.Log("[user-login]", provider, id, name)
}
