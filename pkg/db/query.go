package db

import (
	"strconv"

	"github.com/nektro/go-util/util"

	. "github.com/nektro/go-util/alias"
)

func QueryDoAddUser(id int64, provider string, snowflake string, admin bool, name string) {
	db.Build().Ins("users", id, snowflake, strconv.Itoa(util.Btoi(admin)), name, T(), GenerateNewUserPasskey(snowflake), provider).Exe()
}

func GenerateNewUserPasskey(snowflake string) string {
	return util.Hash("MD5", []byte(F("astheno.andesite.passkey.%s.%s", snowflake, T())))[0:10]
}

func QueryAssertUserName(provider, snowflake string, name string) {
	u, ok := User{}.BySnowflake(provider, snowflake)
	if ok {
		db.Build().Up(ctUser, "provider", provider).Wh("snowflake", snowflake).Exe()
		u.SetName(name)
	} else {
		uid := db.QueryNextID("users")
		QueryDoAddUser(uid, provider, snowflake, false, name)

		if uid == 1 {
			// always admin first user
			db.Build().Up("users", "admin", "1").Wh("id", "1").Exe()
			aid := db.QueryNextID("access")
			db.Build().Ins("access", aid, uid, "/").Exe()
			util.Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}
