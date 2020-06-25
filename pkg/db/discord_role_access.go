package db

import (
	"database/sql"
	"strconv"

	dbstorage "github.com/nektro/go.dbstorage"
)

type DiscordRoleAccess struct {
	ID        int64 `json:"id"`
	IDS       string
	GuildID   string `json:"guild_snowflake" sqlite:"text"`
	RoleID    string `json:"role_snowflake" sqlite:"text"`
	Path      string `json:"path" sqlite:"text"`
	GuildName string `json:"guild_name" sqlite:"text"`
	RoleName  string `json:"role_name" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v DiscordRoleAccess) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return &v
}

func (DiscordRoleAccess) ScanAll(q dbstorage.QueryBuilder) []*DiscordRoleAccess {
	arr := dbstorage.ScanAll(q, DiscordRoleAccess{})
	res := []*DiscordRoleAccess{}
	for _, item := range arr {
		o, ok := item.(*DiscordRoleAccess)
		if !ok {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (DiscordRoleAccess) b() dbstorage.QueryBuilder {
	return DB.Build().Se("*").Fr(ctDiscordRoleAccess)
}

func (DiscordRoleAccess) All() []*DiscordRoleAccess {
	return DiscordRoleAccess{}.ScanAll(DiscordRoleAccess{}.b())
}

//
// searchers
//

func (DiscordRoleAccess) ByID(id int64) (*DiscordRoleAccess, bool) {
	ur, ok := dbstorage.ScanFirst(DiscordRoleAccess{}.b().Wh("id", strconv.FormatInt(id, 10)), DiscordRoleAccess{}).(*DiscordRoleAccess)
	return ur, ok
}

//
// modifiers
//
