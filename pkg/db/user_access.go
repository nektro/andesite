package db

import (
	"database/sql"

	dbstorage "github.com/nektro/go.dbstorage"
)

type UserAccess struct {
	ID   int64  `json:"id"`
	User int64  `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v UserAccess) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.User, &v.Path)
	return &v
}
