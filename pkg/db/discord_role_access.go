package db

import (
	"database/sql"
	"strconv"

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

func CreateDiscordRoleAccess(gi, ri, pt, gn, rn string) *DiscordRoleAccess {
	dbstorage.InsertsLock.Lock()
	defer dbstorage.InsertsLock.Unlock()
	//
	id := db.QueryNextID(ctDiscordRoleAccess)
	rv := &DiscordRoleAccess{id, gi, ri, pt, gn, rn}
	db.Build().InsI(ctDiscordRoleAccess, rv).Exe()
	return rv
}

// Scan implements dbstorage.Scannable
func (v DiscordRoleAccess) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
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

func (v *DiscordRoleAccess) i() string {
	return strconv.FormatInt(v.ID, 10)
}

func (DiscordRoleAccess) b() dbstorage.QueryBuilder {
	return db.Build().Se("*").Fr(ctDiscordRoleAccess)
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

func (v *DiscordRoleAccess) SetGuildID(s string) {
	v.GuildID = s
	Up(v, db, ctDiscordRoleAccess, "guild_snowflake", s)
}

func (v *DiscordRoleAccess) SetRoleID(s string) {
	v.RoleID = s
	Up(v, db, ctDiscordRoleAccess, "role_snowflake", s)
}

func (v *DiscordRoleAccess) SetPath(s string) {
	v.Path = s
	Up(v, db, ctDiscordRoleAccess, "path", s)
}

func (v *DiscordRoleAccess) SetGuildName(s string) {
	v.GuildName = s
	Up(v, db, ctDiscordRoleAccess, "guild_name", s)
}

func (v *DiscordRoleAccess) SetRoleName(s string) {
	v.RoleName = s
	Up(v, db, ctDiscordRoleAccess, "role_name", s)
}

func (v *DiscordRoleAccess) Delete() {
	Del(v, db, ctDiscordRoleAccess)
}
