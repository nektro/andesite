package itypes

import (
	"database/sql"
)

type UserAccess struct {
	ID   int64  `json:"id"`
	User int64  `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}

func ScanUserAccess(rows *sql.Rows) UserAccess {
	var v UserAccess
	rows.Scan(&v.ID, &v.User, &v.Path)
	return v
}
