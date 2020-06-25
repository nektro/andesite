package db

import (
	"database/sql"
	"strconv"

	dbstorage "github.com/nektro/go.dbstorage"
)

type UserAccess struct {
	ID   int64 `json:"id"`
	IDS  string
	User int64  `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v UserAccess) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.User, &v.Path)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return &v
}
