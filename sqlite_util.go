package main

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

func scanUser(rows *sql.Rows) itypes.UserRow {
	var v itypes.UserRow
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey)
	return v
}

func scanAccessRow(rows *sql.Rows) itypes.UserAccessRow {
	var v itypes.UserAccessRow
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}

func scanShare(rows *sql.Rows) itypes.ShareRow {
	var v itypes.ShareRow
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return v
}

//
//

func queryAccess(user itypes.UserRow) []string {
	result := []string{}
	rows := etc.Database.Query(false, F("select * from access where user = '%d'", user.ID))
	for rows.Next() {
		result = append(result, scanAccessRow(rows).Path)
	}
	rows.Close()
	return result
}

func queryUserBySnowflake(snowflake string) (itypes.UserRow, bool) {
	rows := etc.Database.Query(false, F("select * from users where snowflake = '%s'", oauth2Provider.DbP+snowflake))
	if !rows.Next() {
		return itypes.UserRow{}, false
	}
	ur := scanUser(rows)
	rows.Close()
	ur.Snowflake = ur.Snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryUserByID(id int) (itypes.UserRow, bool) {
	rows := etc.Database.Query(false, F("select * from users where id = '%d'", id))
	if !rows.Next() {
		return itypes.UserRow{}, false
	}
	ur := scanUser(rows)
	rows.Close()
	ur.Snowflake = ur.Snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryAllAccess() []map[string]string {
	var result []map[string]string
	rows := etc.Database.Query(false, "select * from access")
	accs := []itypes.UserAccessRow{}
	for rows.Next() {
		accs = append(accs, scanAccessRow(rows))
	}
	rows.Close()
	ids := map[int][]string{}
	for _, uar := range accs {
		if _, ok := ids[uar.User]; !ok {
			uu, _ := queryUserByID(uar.User)
			ids[uar.User] = []string{uu.Snowflake, uu.Name}
		}
		result = append(result, map[string]string{
			"id":        strconv.Itoa(uar.ID),
			"user":      strconv.Itoa(uar.User),
			"snowflake": ids[uar.User][0],
			"name":      ids[uar.User][1],
			"path":      uar.Path,
		})
	}
	return result
}

func queryDoAddUser(id int, snowflake string, admin bool, name string) {
	etc.Database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?, '%s', '')", id, oauth2Provider.DbP+snowflake, boolToString(admin), T()), name)
	etc.Database.QueryDoUpdate("users", "passkey", generateNewUserPasskey(snowflake), "snowflake", snowflake)
}

func queryDoUpdate(table string, col string, value string, where string, search string) {
	etc.Database.QueryPrepared(true, F("update %s set %s = ? where %s = ?", table, col, where), value, search)
}

func queryAssertUserName(snowflake string, name string) {
	_, ok := queryUserBySnowflake(snowflake)
	if ok {
		queryDoUpdate("users", "name", name, "snowflake", oauth2Provider.DbP+snowflake)
	} else {
		uid := etc.Database.QueryNextID("users")
		queryDoAddUser(uid, snowflake, false, name)

		if uid == 0 {
			// always admin first user
			etc.Database.QueryDoUpdate("users", "admin", "1", "id", "0")
			aid := etc.Database.QueryNextID("access")
			etc.Database.Query(true, F("insert into access values ('%d', '%d', '/')", aid, uid))
			Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}

func queryAllShares() []map[string]string {
	var result []map[string]string
	rows := etc.Database.QueryDoSelectAll("shares")
	for rows.Next() {
		sr := scanShare(rows)
		result = append(result, map[string]string{
			"id":   strconv.Itoa(sr.ID),
			"hash": sr.Hash,
			"path": sr.Path,
		})
	}
	rows.Close()
	return result
}

func queryAllSharesByCode(code string) []itypes.ShareRow {
	shrs := []itypes.ShareRow{}
	rows := etc.Database.QueryDoSelect("shares", "hash", code)
	for rows.Next() {
		shrs = append(shrs, scanShare(rows))
	}
	rows.Close()
	return shrs
}

func queryAccessByShare(code string) []string {
	result := []string{}
	for _, item := range queryAllSharesByCode(code) {
		result = append(result, item.Path)
	}
	return result
}

func queryAllDiscordRoleAccess() []itypes.DiscordRoleAccessRow {
	var result []itypes.DiscordRoleAccessRow
	rows := etc.Database.QueryDoSelectAll("shares_discord_role")
	for rows.Next() {
		var v itypes.DiscordRoleAccessRow
		rows.Scan(&v.ID, &v.GuildID, &v.RoleID, &v.Path, &v.GuildName, &v.RoleName)
		result = append(result, v)
	}
	rows.Close()
	return result
}

func queryDiscordRoleAccess(id string) *itypes.DiscordRoleAccessRow {
	sid, _ := strconv.Atoi(id)
	for _, item := range queryAllDiscordRoleAccess() {
		if item.ID == sid {
			return &item
		}
	}
	return nil
}
