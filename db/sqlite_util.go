package db

import (
	"database/sql"
	"strconv"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	"github.com/nektro/andesite/util"
)

//
//

func ScanUser(rows *sql.Rows) UserRow {
	var v UserRow
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey, &v.Provider)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return v
}

func ScanAccessRow(rows *sql.Rows) UserAccessRow {
	var v UserAccessRow
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}

func ScanShare(rows *sql.Rows) ShareRow {
	var v ShareRow
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return v
}

//
//

func QueryAccess(user *UserRow) []string {
	result := []string{}
	rows := etc.Database.Build().Se("*").Fr("access").Wh("user", user.IDS).Exe()
	for rows.Next() {
		result = append(result, ScanAccessRow(rows).Path)
	}
	rows.Close()
	return result
}

func QueryUserBySnowflake(provider, snowflake string) (*UserRow, bool) {
	rows := etc.Database.Build().Se("*").Fr("users").Wh("provider", provider).Wh("snowflake", snowflake).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return &ur, true
}

func QueryUserByID(id int64) (*UserRow, bool) {
	rows := etc.Database.Build().Se("*").Fr("users").Wh("id", strconv.FormatInt(id, 10)).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return &ur, true
}

func QueryAllAccess() []map[string]interface{} {
	var result []map[string]interface{}
	rows := etc.Database.Build().Se("*").Fr("access").Exe()
	accs := []UserAccessRow{}
	for rows.Next() {
		accs = append(accs, ScanAccessRow(rows))
	}
	rows.Close()
	for _, uar := range accs {
		uu, _ := QueryUserByID(uar.User)
		result = append(result, map[string]interface{}{
			"id":    strconv.FormatInt(uar.ID, 10),
			"user":  strconv.FormatInt(uar.User, 10),
			"userO": uu,
			"path":  uar.Path,
		})
	}
	return result
}

func QueryDoAddUser(id int64, provider string, snowflake string, admin bool, name string) {
	etc.Database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?, '%s', '', ?)", id, snowflake, util.BoolToString(admin), T()), name, provider)
	etc.Database.Build().Up("users", "passkey", util.GenerateNewUserPasskey(snowflake)).Wh("snowflake", snowflake).Exe()
}

func QueryAssertUserName(provider, snowflake string, name string) {
	_, ok := QueryUserBySnowflake(provider, snowflake)
	if ok {
		etc.Database.Build().Up("users", "provider", provider).Wh("snowflake", snowflake).Exe()
		etc.Database.Build().Up("users", "name", name).Wh("snowflake", snowflake).Exe()
	} else {
		uid := etc.Database.QueryNextID("users")
		QueryDoAddUser(uid, provider, snowflake, false, name)

		if uid == 1 {
			// always admin first user
			etc.Database.Build().Up("users", "admin", "1").Wh("id", "1").Exe()
			aid := etc.Database.QueryNextID("access")
			etc.Database.QueryPrepared(true, F("insert into access values ('%d', '%d', '/')", aid, uid))
			Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}

func QueryAllShares() []map[string]string {
	var result []map[string]string
	rows := etc.Database.Build().Se("*").Fr("shares").Exe()
	for rows.Next() {
		sr := ScanShare(rows)
		result = append(result, map[string]string{
			"id":   strconv.FormatInt(sr.ID, 10),
			"hash": sr.Hash,
			"path": sr.Path,
		})
	}
	rows.Close()
	return result
}

func QueryAllSharesByCode(code string) []ShareRow {
	shrs := []ShareRow{}
	rows := etc.Database.Build().Se("*").Fr("shares").Wh("hash", code).Exe()
	for rows.Next() {
		shrs = append(shrs, ScanShare(rows))
	}
	rows.Close()
	return shrs
}

func QueryAccessByShare(code string) string {
	result := ""
	for _, item := range QueryAllSharesByCode(code) {
		result = item.Path
	}
	return result
}

func QueryAllDiscordRoleAccess() []DiscordRoleAccessRow {
	var result []DiscordRoleAccessRow
	rows := etc.Database.Build().Se("*").Fr("shares_discord_role").Exe()
	for rows.Next() {
		var v DiscordRoleAccessRow
		rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
		result = append(result, v)
	}
	rows.Close()
	return result
}

func QueryDiscordRoleAccess(id int64) *DiscordRoleAccessRow {
	for _, item := range QueryAllDiscordRoleAccess() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func QueryAllUsers() []UserRow {
	result := []UserRow{}
	q := etc.Database.Build().Se("*").Fr("users").Exe()
	for q.Next() {
		result = append(result, ScanUser(q))
	}
	return result
}
