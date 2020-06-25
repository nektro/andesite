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

func (User) ScanAll(q dbstorage.QueryBuilder) []*User {
	arr := dbstorage.ScanAll(q, User{})
	res := []*User{}
	for _, item := range arr {
		o, ok := item.(*User)
		if !ok {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (User) b() dbstorage.QueryBuilder {
	return DB.Build().Se("*").Fr(ctUser)
}

func (User) All() []*User {
	return User{}.ScanAll(User{}.b())
}

//
// searchers
//

func (User) BySnowflake(provider, snowflake string) (*User, bool) {
	ur, ok := dbstorage.ScanFirst(User{}.b().Wh("provider", provider).Wh("snowflake", snowflake), User{}).(*User)
	return ur, ok
}

func (User) ByPasskey(pk string) (*User, bool) {
	ur, ok := dbstorage.ScanFirst(User{}.b().Wh("passkey", pk), User{}).(*User)
	return ur, ok
}

func (User) ByID(id int64) (*User, bool) {
	ur, ok := dbstorage.ScanFirst(User{}.b().Wh("id", strconv.FormatInt(id, 10)), User{}).(*User)
	return ur, ok
}

//
// modifiers
//

func (v *User) GetAccess() []string {
	res := []string{}
	arr := UserAccess{}.ByUser(v)
	for _, item := range arr {
		res = append(res, item.Path)
	}
	return res
}

func (v *User) FullName() string {
	return v.Name + "@" + v.Provider
}
