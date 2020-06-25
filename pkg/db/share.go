package db

import (
	"database/sql"
	"strconv"

	dbstorage "github.com/nektro/go.dbstorage"
)

type Share struct {
	ID   int64 `json:"id"`
	IDS  string
	Hash string `json:"hash" sqlite:"text"`
	Path string `json:"path" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v Share) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return &v
}
