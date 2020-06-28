package handler

import (
	"net/http"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"

	. "github.com/nektro/go-util/alias"
)

func HandleDiscordRoleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	if !ContainsAll(r.PostForm, "RoleID", "Path") {
		WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	// ags := r.PostForm.Get("GuildID")
	ags := idata.Config.GetDiscordClient().Extra1
	agr := r.PostForm.Get("RoleID")
	apt := r.PostForm.Get("Path")
	//
	gn := FetchDiscordGuild(ags).Name
	rn := FetchDiscordRole(ags, agr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	db.CreateDiscordRoleAccess(ags, agr, apt, gn, rn)
	WriteAPIResponse(r, w, true, F("Created access for %s / %s to %s.", gn, rn, apt))
}

func HandleDiscordRoleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	//
	if !ContainsAll(r.PostForm, "ID", "RoleID", "Path") {
		WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	_, qid, err := hGrabID(r, w)
	if err != nil {
		return
	}
	// qgs := r.PostForm.Get("GuildID")
	qgs := idata.Config.GetDiscordClient().Extra1
	qgr := r.PostForm.Get("RoleID")
	qpt := r.PostForm.Get("Path")
	//
	gn := FetchDiscordGuild(qgs).Name
	rn := FetchDiscordRole(qgs, qgr).Name
	//
	if len(gn) == 0 && len(rn) == 0 {
		WriteAPIResponse(r, w, false, "Unable to fetch role metadata from Discord API.")
		return
	}
	dra, ok := db.DiscordRoleAccess{}.ByID(qid)
	if !ok {
		return
	}
	dra.SetGuildID(qgs)
	dra.SetGuildName(gn)
	dra.SetRoleID(qgr)
	dra.SetRoleName(rn)
	dra.SetPath(qpt)
	WriteAPIResponse(r, w, true, F("Successfully updated share path for %s / %s to %s.", gn, rn, qpt))
}

func HandleDiscordRoleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, err := ApiBootstrap(r, w, []string{http.MethodPost}, true, true, true)
	if err != nil {
		return
	}
	if !ContainsAll(r.PostForm, "ID") {
		WriteAPIResponse(r, w, false, "Missing POST values")
		return
	}
	_, qID, err := hGrabID(r, w)
	if err != nil {
		return
	}
	dra, ok := db.DiscordRoleAccess{}.ByID(qID)
	if !ok {
		return
	}
	dra.Delete()
	WriteAPIResponse(r, w, true, F("Successfully deleted access for %s / %s to %s.", dra.GuildName, dra.RoleName, dra.Path))
}
