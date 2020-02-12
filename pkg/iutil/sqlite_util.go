package iutil

import (
	"database/sql"
	"strconv"

	"github.com/nektro/andesite/pkg/itypes"

	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"
)

//
//

func ScanUser(rows *sql.Rows) itypes.User {
	var v itypes.User
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey, &v.Provider)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return v
}

func ScanUserAccess(rows *sql.Rows) itypes.UserAccess {
	var v itypes.UserAccess
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}

func ScanShare(rows *sql.Rows) itypes.Share {
	var v itypes.Share
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return v
}

//
//

func QueryAccess(user *itypes.User) []string {
	result := []string{}
	rows := etc.Database.Build().Se("*").Fr("access").Wh("user", user.IDS).Exe()
	for rows.Next() {
		result = append(result, ScanUserAccess(rows).Path)
	}
	rows.Close()
	return result
}

func QueryUserBySnowflake(provider, snowflake string) (*itypes.User, bool) {
	rows := etc.Database.Build().Se("*").Fr("users").Wh("provider", provider).Wh("snowflake", snowflake).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return &ur, true
}

func QueryUserByID(id int64) (*itypes.User, bool) {
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
	accs := []itypes.UserAccess{}
	for rows.Next() {
		accs = append(accs, ScanUserAccess(rows))
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
	etc.Database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?, '%s', '', ?)", id, snowflake, BoolToString(admin), T()), name, provider)
	etc.Database.Build().Up("users", "passkey", GenerateNewUserPasskey(snowflake)).Wh("snowflake", snowflake).Exe()
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
			util.Log(F("Set user '%s's status to admin", snowflake))
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

func QueryAllSharesByCode(code string) []itypes.Share {
	shrs := []itypes.Share{}
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

func QueryAllDiscordRoleAccess() []itypes.DiscordRoleAccess {
	var result []itypes.DiscordRoleAccess
	rows := etc.Database.Build().Se("*").Fr("shares_discord_role").Exe()
	for rows.Next() {
		var v itypes.DiscordRoleAccess
		rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
		result = append(result, v)
	}
	rows.Close()
	return result
}

func QueryDiscordRoleAccess(id int64) *itypes.DiscordRoleAccess {
	for _, item := range QueryAllDiscordRoleAccess() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func QueryAllUsers() []itypes.User {
	result := []itypes.User{}
	q := etc.Database.Build().Se("*").Fr("users").Exe()
	for q.Next() {
		result = append(result, ScanUser(q))
	}
	return result
}
