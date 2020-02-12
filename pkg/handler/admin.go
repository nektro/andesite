package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"
)

// handler for http://andesite/admin
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrap(r, w, []string{http.MethodGet}, true, true, true)
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
		"users":                 db.QueryAllUsers(),
		"accesses":              db.QueryAllAccess(),
		"shares":                db.QueryAllShares(),
		"discord_shares":        db.QueryAllDiscordRoleAccess(),
	})
}

// handler for http://andesite/admin/users
func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrap(r, w, []string{http.MethodGet}, true, true, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/users.hbs", map[string]interface{}{
		"version": idata.Version,
		"user":    user,
		"base":    idata.Config.HTTPBase,
		"name":    oauth2.ProviderIDMap[user.Provider].NamePrefix + user.Name,
		"auth":    oauth2.ProviderIDMap[user.Provider].ID,
		"users":   db.QueryAllUsers(),
	})
}

func HandleAdminRoots(w http.ResponseWriter, r *http.Request) {
	_, user, err := iutil.ApiBootstrap(r, w, []string{http.MethodGet}, true, true, true)
	if err != nil {
		return
	}
	etc.WriteHandlebarsFile(r, w, "/admin_roots.hbs", map[string]interface{}{
		"version":       idata.Version,
		"user":          user,
		"base":          idata.Config.HTTPBase,
		"roots_public":  iutil.MapToArray(idata.DataPathsPub),
		"roots_private": iutil.MapToArray(idata.DataPathsPrv),
	})
}
