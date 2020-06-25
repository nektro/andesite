package db

import (
	"database/sql"

	dbstorage "github.com/nektro/go.dbstorage"
)

type Share struct {
	ID   int64  `json:"id"`
	Hash string `json:"hash" sqlite:"text"` // character(32)
	Path string `json:"path" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v Share) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return &v
}
