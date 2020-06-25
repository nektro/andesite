package db

import (
	"database/sql"

	dbstorage "github.com/nektro/go.dbstorage"
)

type DiscordRoleAccess struct {
	ID        int64  `json:"id"`
	GuildID   string `json:"guild_snowflake" sqlite:"text"`
	RoleID    string `json:"role_snowflake" sqlite:"text"`
	Path      string `json:"path" sqlite:"text"`
	GuildName string `json:"guild_name" sqlite:"text"`
	RoleName  string `json:"role_name" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v DiscordRoleAccess) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
	return &v
}
