package main

import (
	"database/sql"
	"fmt"
	"strconv"
)

func createTable(name string, pk []string, columns [][]string) {
	if !doesTableExist(name) {
		// create table
		query(fmt.Sprintf("create table %s(%s %s)", name, pk[0], pk[1]), true)
		log(fmt.Sprintf("Created table '%s'", name))
	}
	// add rows
	pti := queryColumnList(name)
	for _, col := range columns {
		if !contains(pti, col[0]) {
			query(fmt.Sprintf("alter table %s add %s %s", name, col[0], col[1]), true)
			log(fmt.Sprintf("Added column '%s' to table '%s'", col[0], name))
		}
	}
}

func doesTableExist(table string) bool {
	q := query(fmt.Sprintf("select name from sqlite_master where type='table' AND name='%s';", table), false)
	e := q.Next()
	q.Close()
	return e
}

func query(q string, modify bool) *sql.Rows {
	if modify {
		_, err := database.Exec(q)
		checkErr(err)
		return nil
	}
	rows, err := database.Query(q)
	checkErr(err)
	return rows
}

func queryColumnList(table string) []string {
	var result []string
	rows := query(fmt.Sprintf("pragma table_info(%s)", table), false)
	for rows.Next() {
		var cid int
		var name string
		var typeV string
		var notnull bool
		var dfltValue interface{}
		var pk int
		rows.Scan(&cid, &name, &typeV, &notnull, &dfltValue, &pk)
		result = append(result, name)
	}
	rows.Close()
	return result
}

func queryAccess(snowflake string) []string {
	result := []string{}
	user, ok := queryUserBySnowflake(snowflake)
	if !ok {
		return result
	}
	rows := query(fmt.Sprintf("select * from access where user = '%d'", user.id), false)
	for rows.Next() {
		var id int
		var user int
		var path string
		rows.Scan(&id, &user, &path)
		result = append(result, path)
	}
	rows.Close()
	return result
}

func queryUserBySnowflake(snowflake string) (UserRow, bool) {
	var ur UserRow
	rows := query(fmt.Sprintf("select * from users where snowflake = '%s'", snowflake), false)
	if !rows.Next() {
		return ur, false
	}
	rows.Scan(&ur.id, &ur.snowflake, &ur.admin)
	rows.Close()
	return ur, true
}

func queryUserByID(id int) (UserRow, bool) {
	var ur UserRow
	rows := query(fmt.Sprintf("select * from users where id = '%d'", id), false)
	if !rows.Next() {
		return ur, false
	}
	rows.Scan(&ur.id, &ur.snowflake, &ur.admin)
	rows.Close()
	return ur, true
}

func queryAllAccess() []map[string]string {
	var result []map[string]string
	rows := query("select * from access", false)
	ids := map[int]string{}
	for rows.Next() {
		var uar UserAccessRow
		rows.Scan(&uar.id, &uar.user, &uar.path)
		if _, ok := ids[uar.user]; !ok {
			uu, _ := queryUserByID(uar.user)
			ids[uar.user] = uu.snowflake
		}
		result = append(result, map[string]string{
			"id":        strconv.Itoa(uar.id),
			"user":      strconv.Itoa(uar.user),
			"snowflake": ids[uar.user],
			"path":      uar.path,
		})
	}
	rows.Close()
	return result
}

func queryLastID(table string) int {
	result := -1
	rows := query(fmt.Sprintf("select id from %s order by id desc limit 1", table), false)
	for rows.Next() {
		rows.Scan(&result)
	}
	rows.Close()
	return result
}

func queryPrepared(q string, modify bool, args ...interface{}) *sql.Rows {
	stmt, err := database.Prepare(q)
	if modify {
		_, err := stmt.Exec(args...)
		checkErr(err)
		return nil
	}
	rows, err := stmt.Query(args...)
	checkErr(err)
	return rows
}
