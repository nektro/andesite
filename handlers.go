package main

import (
	"net/http"

	"github.com/nektro/andesite/pkg/iutil"

	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"
)

func helperOA2SaveInfo(w http.ResponseWriter, r *http.Request, provider string, id string, name string, resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["provider"] = provider
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Values[provider+"_access_token"] = resp["access_token"]
	sess.Values[provider+"_expires_in"] = resp["expires_in"]
	sess.Values[provider+"_refresh_token"] = resp["refresh_token"]
	sess.Save(r, w)
	iutil.QueryAssertUserName(provider, id, name)
	util.Log("[user-login]", provider, id, name)
}
