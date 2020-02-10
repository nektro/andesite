package db

import (
	"strings"

	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/util"
)

var (
	Prefixes = map[string]string{
		"reddit":    "1:",
		"github":    "2:",
		"google":    "3:",
		"facebook":  "4:",
		"microsoft": "5:",
	}
)

func Init() {
	etc.Database.CreateTableStruct("users", UserRow{})
	etc.Database.CreateTableStruct("access", UserAccessRow{})
	etc.Database.CreateTableStruct("shares", ShareRow{})
	etc.Database.CreateTableStruct("shares_discord_role", DiscordRoleAccessRow{})
}

func Upgrade() {
	for _, item := range QueryAllUsers() {
		for k, v := range Prefixes {
			if strings.HasPrefix(item.Snowflake, v) {
				sn := item.Snowflake[len(v):]
				Log("[db-upgrade]", item.Snowflake, "is now", sn, "as", k)
				etc.Database.Build().Up("users", "snowflake", sn).Wh("id", item.IDS).Exe()
				etc.Database.Build().Up("users", "provider", k).Wh("id", item.IDS).Exe()
			}
		}
	}
}
