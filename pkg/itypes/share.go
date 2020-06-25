package itypes

import (
	"database/sql"
)

type Share struct {
	ID   int64  `json:"id"`
	Hash string `json:"hash" sqlite:"text"` // character(32)
	Path string `json:"path" sqlite:"text"`
}

func ScanShare(rows *sql.Rows) *Share {
	var v Share
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return &v
}
