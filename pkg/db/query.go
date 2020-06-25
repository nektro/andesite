package db

import (
	"strconv"

	"github.com/nektro/go-util/util"

	. "github.com/nektro/go-util/alias"
)

func QueryAccess(user *User) []string {
	result := []string{}
	rows := DB.Build().Se("*").Fr("access").Wh("user", user.IDS).Exe()
	for rows.Next() {
		result = append(result, ScanUserAccess(rows).Path)
	}
	rows.Close()
	return result
}

func QueryUserBySnowflake(provider, snowflake string) (*User, bool) {
	rows := DB.Build().Se("*").Fr("users").Wh("provider", provider).Wh("snowflake", snowflake).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return ur, true
}

func QueryUserByID(id int64) (*User, bool) {
	rows := DB.Build().Se("*").Fr("users").Wh("id", strconv.FormatInt(id, 10)).Exe()
	if !rows.Next() {
		return nil, false
	}
	ur := ScanUser(rows)
	rows.Close()
	return ur, true
}

func QueryAllAccess() []map[string]interface{} {
	var result []map[string]interface{}
	rows := DB.Build().Se("*").Fr("access").Exe()
	accs := []*UserAccess{}
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
	DB.Build().Ins("users", id, snowflake, strconv.Itoa(util.Btoi(admin)), name, T(), GenerateNewUserPasskey(snowflake), provider).Exe()
}

func GenerateNewUserPasskey(snowflake string) string {
	return util.Hash("MD5", []byte(F("astheno.andesite.passkey.%s.%s", snowflake, T())))[0:10]
}

func QueryAssertUserName(provider, snowflake string, name string) {
	_, ok := QueryUserBySnowflake(provider, snowflake)
	if ok {
		DB.Build().Up(ctUser, "provider", provider).Wh("snowflake", snowflake).Exe()
		DB.Build().Up(ctUser, "name", name).Wh("snowflake", snowflake).Exe()
	} else {
		uid := DB.QueryNextID("users")
		QueryDoAddUser(uid, provider, snowflake, false, name)

		if uid == 1 {
			// always admin first user
			DB.Build().Up("users", "admin", "1").Wh("id", "1").Exe()
			aid := DB.QueryNextID("access")
			DB.Build().Ins("access", aid, uid, "/").Exe()
			util.Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}

func QueryAllShares() []map[string]string {
	var result []map[string]string
	rows := DB.Build().Se("*").Fr("shares").Exe()
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

func QueryAllSharesByCode(code string) []*Share {
	shrs := []*Share{}
	rows := DB.Build().Se("*").Fr("shares").Wh("hash", code).Exe()
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

func QueryAllDiscordRoleAccess() []DiscordRoleAccess {
	var result []DiscordRoleAccess
	rows := DB.Build().Se("*").Fr("shares_discord_role").Exe()
	for rows.Next() {
		var v DiscordRoleAccess
		rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
		result = append(result, v)
	}
	rows.Close()
	return result
}

func QueryDiscordRoleAccess(id int64) *DiscordRoleAccess {
	for _, item := range QueryAllDiscordRoleAccess() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func QueryAllUsers() []*User {
	result := []*User{}
	q := DB.Build().Se("*").Fr("users").Exe()
	for q.Next() {
		result = append(result, ScanUser(q))
	}
	return result
}
