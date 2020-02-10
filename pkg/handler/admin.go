package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
)

// handler for http://andesite/admin
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	dc := idata.Config.GetDiscordClient()
	etc.WriteHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"version":               idata.Version,
		"user":                  user,
		"base":                  idata.Config.HTTPBase,
		"name":                  oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":                  oauth2.ProviderIDMap[user.Provider].ID,
		"discord_role_share_on": len(dc.Extra1) > 0 && len(dc.Extra2) > 0,
		"users":                 iutil.QueryAllUsers(),
		"accesses":              iutil.QueryAllAccess(),
		"shares":                iutil.QueryAllShares(),
		"discord_shares":        iutil.QueryAllDiscordRoleAccess(),
	})
}

// handler for http://andesite/admin/users
func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/users.hbs", map[string]interface{}{
		"version": idata.Version,
		"user":    user,
		"base":    idata.Config.HTTPBase,
		"name":    oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":    oauth2.ProviderIDMap[user.Provider].ID,
		"users":   iutil.QueryAllUsers(),
	})
}
