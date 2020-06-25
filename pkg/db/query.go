package db

import (
	"strconv"

	"github.com/nektro/go-util/util"

	. "github.com/nektro/go-util/alias"
)

func QueryDoAddUser(id int64, provider string, snowflake string, admin bool, name string) {
	DB.Build().Ins("users", id, snowflake, strconv.Itoa(util.Btoi(admin)), name, T(), GenerateNewUserPasskey(snowflake), provider).Exe()
}

func GenerateNewUserPasskey(snowflake string) string {
	return util.Hash("MD5", []byte(F("astheno.andesite.passkey.%s.%s", snowflake, T())))[0:10]
}

func QueryAssertUserName(provider, snowflake string, name string) {
	_, ok := User{}.BySnowflake(provider, snowflake)
	if ok {
		DB.Build().Up(ctUser, "provider", provider).Wh("snowflake", snowflake).Exe()
		DB.Build().Up(ctUser, "name", name).Wh("snowflake", snowflake).Exe()
	} else {
		uid := DB.QueryNextID("users")
		QueryDoAddUser(uid, provider, snowflake, false, name)

		if uid == 1 {
			// always admin first user
			DB.Build().Up("users", "admin", "1").Wh("id", "1").Exe()
			aid := DB.QueryNextID("access")
			DB.Build().Ins("access", aid, uid, "/").Exe()
			util.Log(F("Set user '%s's status to admin", snowflake))
		}
	}
}
