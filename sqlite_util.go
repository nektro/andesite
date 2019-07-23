package main

import (
	"database/sql"
	"strconv"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"
)

func scanUser(rows *sql.Rows) UserRow {
	var v UserRow
	rows.Scan(&v.id, &v.snowflake, &v.admin, &v.name)
	return v
}

func scanAccessRow(rows *sql.Rows) UserAccessRow {
	var v UserAccessRow
	rows.Scan(&v.id, &v.user, &v.path)
	return v
}

//
//

func queryAccess(user UserRow) []string {
	result := []string{}
	rows := database.Query(false, F("select * from access where user = '%d'", user.id))
	for rows.Next() {
		result = append(result, scanAccessRow(rows).path)
	}
	rows.Close()
	return result
}

func queryUserBySnowflake(snowflake string) (UserRow, bool) {
	var ur UserRow
	rows := database.Query(false, F("select * from users where snowflake = '%s'", oauth2Provider.DbP+snowflake))
	if !rows.Next() {
		return ur, false
	}
	ur = scanUser(rows)
	rows.Close()
	ur.snowflake = ur.snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryUserByID(id int) (UserRow, bool) {
	var ur UserRow
	rows := database.Query(false, F("select * from users where id = '%d'", id))
	if !rows.Next() {
		return ur, false
	}
	rows.Scan(&ur.id, &ur.snowflake, &ur.admin, &ur.name)
	rows.Close()
	ur.snowflake = ur.snowflake[len(oauth2Provider.DbP):]
	return ur, true
}

func queryAllAccess() []map[string]string {
	var result []map[string]string
	rows := database.Query(false, "select * from access")
	accs := []UserAccessRow{}
	for rows.Next() {
		accs = append(accs, scanAccessRow(rows))
	}
	rows.Close()
	ids := map[int][]string{}
	for _, uar := range accs {
		if _, ok := ids[uar.user]; !ok {
			uu, _ := queryUserByID(uar.user)
			ids[uar.user] = []string{uu.snowflake, uu.name}
		}
		result = append(result, map[string]string{
			"id":        strconv.Itoa(uar.id),
			"user":      strconv.Itoa(uar.user),
			"snowflake": ids[uar.user][0],
			"name":      ids[uar.user][1],
			"path":      uar.path,
		})
	}
	return result
}

func queryDoAddUser(id int, snowflake string, admin bool, name string) {
	database.QueryPrepared(true, F("insert into users values ('%d', '%s', '%s', ?)", id, oauth2Provider.DbP+snowflake, boolToString(admin)), name)
}

func queryDoUpdate(table string, col string, value string, where string, search string) {
	database.QueryPrepared(true, F("update %s set %s = ? where %s = ?", table, col, where), value, search)
}

func queryAssertUserName(snowflake string, name string) {
	_, ok := queryUserBySnowflake(snowflake)
	if ok {
		queryDoUpdate("users", "name", name, "snowflake", oauth2Provider.DbP+snowflake)
	} else {
		uid := database.QueryNextID("users")
		queryDoAddUser(uid, snowflake, false, name)

		if uid == 0 {
			// always admin first user
			database.QueryDoUpdate("users", "admin", "1", "id", "0")
			aid := database.QueryNextID("access")
			database.Query(true, F("insert into access values ('%d', '%d', '/')", aid, uid))
			Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}

func queryAllShares() []map[string]string {
	var result []map[string]string
	rows := database.Query(false, "select * from shares")
	for rows.Next() {
		var sr ShareRow
		rows.Scan(&sr.id, &sr.hash, &sr.path)
		result = append(result, map[string]string{
			"id":   strconv.Itoa(sr.id),
			"hash": sr.hash,
			"path": sr.path,
		})
	}
	rows.Close()
	return result
}

func queryAllSharesByCode(code string) []ShareRow {
	shrs := []ShareRow{}
	rows := database.QueryPrepared(false, "select * from shares where hash = ?", code)
	for rows.Next() {
		var sr ShareRow
		rows.Scan(&sr.id, &sr.hash, &sr.path)
		shrs = append(shrs, sr)
	}
	rows.Close()
	return shrs
}

func queryAccessByShare(code string) []string {
	result := []string{}
	for _, item := range queryAllSharesByCode(code) {
		result = append(result, item.path)
	}
	return result
}
