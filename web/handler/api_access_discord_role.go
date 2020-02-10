package handler

import (
	"net/http"
	"strconv"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/util"
)

func HandleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !util.ContainsAll(r.PostForm, "RoleID", "Path") {
		util.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := etc.Database.QueryNextID("shares_discord_role")
	// ags := r.PostForm.Get("GuildID")
	ags := config.Config.GetDiscordClient().Extra1
	agr := r.PostForm.Get("RoleID")
	apt := r.PostForm.Get("Path")
	//
	gn := util.FetchDiscordGuild(ags).Name
	rn := util.FetchDiscordRole(ags, agr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		util.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	etc.Database.QueryPrepared(true, "insert into shares_discord_role values (?, ?, ?, ?, ?, ?)", aid, ags, agr, apt, gn, rn)
	util.WriteAPIResponse(r, w, true, F("Created access for %s / %s to %s.", gn, rn, apt))
}

func HandleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	//
	if !util.ContainsAll(r.PostForm, "ID", "RoleID", "Path") {
		util.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	qid := r.PostForm.Get("ID")
	// qgs := r.PostForm.Get("GuildID")
	qgs := config.Config.GetDiscordClient().Extra1
	qgr := r.PostForm.Get("RoleID")
	qpt := r.PostForm.Get("Path")
	//
	gn := util.FetchDiscordGuild(qgs).Name
	rn := util.FetchDiscordRole(qgs, qgr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		util.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	etc.Database.Build().Up("shares_discord_role", "guild_snowflake", qgs).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_snowflake", qgr).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "path", qpt).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "guild_name", gn).Wh("id", qid).Exe()
	etc.Database.Build().Up("shares_discord_role", "role_name", rn).Wh("id", qid).Exe()
	util.WriteAPIResponse(r, w, true, F("Successfully updated share path for %s / %s to %s.", gn, rn, qpt))
}

func HandleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := db.ApiBootstrapRequireLogin(r, w, []string{http.MethodPost}, true)
	if err != nil {
		return
	}
	if !util.ContainsAll(r.PostForm, "ID") {
		util.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	qID, err := strconv.ParseInt(r.PostForm.Get("ID"), 10, 64)
	if err != nil {
		return
	}
	dra := db.QueryDiscordRoleAccess(qID)
	if dra == nil {
		return
	}
	etc.Database.QueryPrepared(true, "delete from shares_discord_role where id = ?", qID)
	util.WriteAPIResponse(r, w, true, F("Successfully deleted access for %s / %s to %s.", dra.GuildName, dra.RoleName, dra.Path))
}
