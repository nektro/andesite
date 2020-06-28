package db

import (
	"database/sql"
	"strconv"

	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"

	. "github.com/nektro/go-util/alias"
)

type User struct {
	ID        int64  `json:"id"`
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

func (v *User) i() string {
	return strconv.FormatInt(v.ID, 10)
}

func (User) b() dbstorage.QueryBuilder {
	return db.Build().Se("*").Fr(ctUser)
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

func (v *User) SetProvider(s string) {
	v.Provider = s
	Up(v, db, ctUser, "provider", s)
}

func (v *User) SetSnowflake(s string) {
	v.Snowflake = s
	Up(v, db, ctUser, "snowflake", s)
}

func (v *User) ResetPasskey() {
	s := util.Hash("MD5", []byte(F("astheno.andesite.passkey.%s.%s", v.Snowflake, T())))[0:10]
	v.PassKey = s
	Up(v, db, ctUser, "passkey", s)
}

func (v *User) SetName(s string) {
	v.Name = s
	Up(v, db, ctUser, "name", s)
}
