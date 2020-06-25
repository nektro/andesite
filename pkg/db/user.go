package db

import (
	"database/sql"
	"strconv"

	dbstorage "github.com/nektro/go.dbstorage"
)

type User struct {
	ID        int64 `json:"id"`
	IDS       string
	Snowflake string `json:"snowflake" sqlite:"text"`
	Admin     bool   `json:"admin" sqlite:"tinyint(1)"`
	Name      string `json:"name" sqlite:"text"`
	JoinedOn  string `json:"joined_on" sqlite:"text"`
	PassKey   string `json:"passkey" sqlite:"text"`
	Provider  string `json:"provider" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v User) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey, &v.Provider)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return &v
}
