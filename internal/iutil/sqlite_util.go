package iutil

import (
	"database/sql"
	"strconv"

	etc "github.com/nektro/go.etc"

	"github.com/nektro/andesite/internal/itypes"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

//
//

func ScanUser(rows *sql.Rows) itypes.UserRow {
	var v itypes.UserRow
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey, &v.Provider)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return v
}

func ScanAccessRow(rows *sql.Rows) itypes.UserAccessRow {
	var v itypes.UserAccessRow
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}

func ScanShare(rows *sql.Rows) itypes.ShareRow {
	var v itypes.ShareRow
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return v
}

//
//

func QueryAccess(user *itypes.UserRow) []string {
	result := []string{}
	rows := etc.Database.Build().Se("*").Fr("access").Wh("user", user.IDS).Exe()
	for rows.Next() {
		result = append(result, ScanAccessRow(rows).Path)
	}
	rows.Close()
	return result
}

func QueryUserBySnowflake(snowflake string) (*itypes.UserRow, bool) {
	rows := etc.Database.Build().Se("*").Fr("users").Wh("snowflake", snowflake).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return &ur, true
}

func QueryUserByID(id int64) (*itypes.UserRow, bool) {
	rows := etc.Database.Build().Se("*").Fr("users").Wh("id", strconv.FormatInt(id, 10)).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return &ur, true
}

func QueryAllAccess() []map[string]string {
	var result []map[string]string
	rows := etc.Database.Build().Se("*").Fr("access").Exe()
	accs := []itypes.UserAccessRow{}
	for rows.Next() {
		accs = append(accs, ScanAccessRow(rows))
	}
	rows.Close()
	ids := map[int64][]string{}
	for _, uar := range accs {
		if _, ok := ids[uar.User]; !ok {
			uu, _ := QueryUserByID(uar.User)
			ids[uar.User] = []string{uu.Snowflake, uu.Name}
		}
		result = append(result, map[string]string{
			"id":        strconv.FormatInt(uar.ID, 10),
			"user":      strconv.FormatInt(uar.User, 10),
			"snowflake": ids[uar.User][0],
			"name":      ids[uar.User][1],
			"path":      uar.Path,
		})
	}
	return result
}

func QueryDoAddUser(id int64, provider string, snowflake string, admin bool, name string) {
	etc.Database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?, '%s', '', ?)", id, snowflake, BoolToString(admin), T()), name, provider)
	etc.Database.Build().Up("users", "passkey", GenerateNewUserPasskey(snowflake)).Wh("snowflake", snowflake).Exe()
}

func QueryAssertUserName(provider string, snowflake string, name string) {
	_, ok := QueryUserBySnowflake(snowflake)
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

func QueryAllSharesByCode(code string) []itypes.ShareRow {
	shrs := []itypes.ShareRow{}
	rows := etc.Database.Build().Se("*").Fr("shares").Wh("hash", code).Exe()
	for rows.Next() {
		shrs = append(shrs, ScanShare(rows))
	}
	rows.Close()
	return shrs
}

func QueryAccessByShare(code string) []string {
	result := []string{}
	for _, item := range QueryAllSharesByCode(code) {
		result = append(result, item.Path)
	}
	return result
}

func QueryAllDiscordRoleAccess() []itypes.DiscordRoleAccessRow {
	var result []itypes.DiscordRoleAccessRow
	rows := etc.Database.Build().Se("*").Fr("shares_discord_role").Exe()
	for rows.Next() {
		var v itypes.DiscordRoleAccessRow
		rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
		result = append(result, v)
	}
	rows.Close()
	return result
}

func QueryDiscordRoleAccess(id int64) *itypes.DiscordRoleAccessRow {
	for _, item := range QueryAllDiscordRoleAccess() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func QueryAllUsers() []itypes.UserRow {
	result := []itypes.UserRow{}
	q := etc.Database.Build().Se("*").Fr("shares_discord_role").Exe()
	for q.Next() {
		result = append(result, ScanUser(q))
	}
	return result
}
