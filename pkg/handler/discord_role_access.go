package handler

import (
	"net/http"
	"strconv"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	. "github.com/nektro/go-util/alias"
)

func HandleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "RoleID", "Path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := db.DB.QueryNextID("shares_discord_role")
	// ags := r.PostForm.Get("GuildID")
	ags := idata.Config.GetDiscordClient().Extra1
	agr := r.PostForm.Get("RoleID")
	apt := r.PostForm.Get("Path")
	//
	gn := iutil.FetchDiscordGuild(ags).Name
	rn := iutil.FetchDiscordRole(ags, agr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		iutil.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	db.DB.Build().Ins("shares_discord_role", aid, ags, agr, apt, gn, rn).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Created access for %s / %s to %s.", gn, rn, apt))
}

func HandleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	if !iutil.ContainsAll(r.PostForm, "ID", "RoleID", "Path") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	qid := r.PostForm.Get("ID")
	// qgs := r.PostForm.Get("GuildID")
	qgs := idata.Config.GetDiscordClient().Extra1
	qgr := r.PostForm.Get("RoleID")
	qpt := r.PostForm.Get("Path")
	//
	gn := iutil.FetchDiscordGuild(qgs).Name
	rn := iutil.FetchDiscordRole(qgs, qgr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		iutil.WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	db.DB.Build().Up("shares_discord_role", "guild_snowflake", qgs).Wh("id", qid).Exe()
	db.DB.Build().Up("shares_discord_role", "role_snowflake", qgr).Wh("id", qid).Exe()
	db.DB.Build().Up("shares_discord_role", "path", qpt).Wh("id", qid).Exe()
	db.DB.Build().Up("shares_discord_role", "guild_name", gn).Wh("id", qid).Exe()
	db.DB.Build().Up("shares_discord_role", "role_name", rn).Wh("id", qid).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Successfully updated share path for %s / %s to %s.", gn, rn, qpt))
}

func HandleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := iutil.ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	if !iutil.ContainsAll(r.PostForm, "ID") {
		iutil.WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	qID, err := strconv.ParseInt(r.PostForm.Get("ID"), 10, 64)
	if err != nil {
		return
	}
	dra, ok := db.DiscordRoleAccess{}.ByID(qID)
	if !ok {
		return
	}
	db.DB.Build().Del("shares_discord_role").Wh("id", strconv.FormatInt(qID, 10)).Exe()
	iutil.WriteAPIResponse(r, w, true, F("Successfully deleted access for %s / %s to %s.", dra.GuildName, dra.RoleName, dra.Path))
}
