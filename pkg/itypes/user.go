package itypes

import (
	"database/sql"
	"strconv"
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

func ScanUser(rows *sql.Rows) *User {
	var v User
	rows.Scan(&v.ID, &v.Snowflake, &v.Admin, &v.Name, &v.JoinedOn, &v.PassKey, &v.Provider)
	v.IDS = strconv.FormatInt(v.ID, 10)
	return &v
}
