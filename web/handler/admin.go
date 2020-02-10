package handler

import (
	"net/http"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
)

// handler for http://andesite/admin
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	dc := config.Config.GetDiscordClient()
	etc.WriteHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"version":               config.Version,
		"user":                  user,
		"base":                  config.Config.HTTPBase,
		"name":                  oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":                  oauth2.ProviderIDMap[user.Provider].ID,
		"discord_role_share_on": len(dc.Extra1) > 0 && len(dc.Extra2) > 0,
		"users":                 db.QueryAllUsers(),
		"accesses":              db.QueryAllAccess(),
		"shares":                db.QueryAllShares(),
		"discord_shares":        db.QueryAllDiscordRoleAccess(),
	})
}

func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, user, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodGet}, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/users.hbs", map[string]interface{}{
		"version": config.Version,
		"user":    user,
		"base":    config.Config.HTTPBase,
		"name":    oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":    oauth2.ProviderIDMap[user.Provider].ID,
		"users":   db.QueryAllUsers(),
	})
}
