package db

import (
	"strings"

	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"
	etc "github.com/nektro/go.etc"
)

var (
	DB dbstorage.Database
)

func Init() {
	DB = etc.Database
	DB.CreateTableStruct("users", itypes.UserRow{})
	DB.CreateTableStruct("access", itypes.UserAccessRow{})
	DB.CreateTableStruct("shares", itypes.ShareRow{})
	DB.CreateTableStruct("shares_discord_role", itypes.DiscordRoleAccessRow{})
}

func Upgrade() {
	prefixes := map[string]string{
		"reddit":    "1:",
		"github":    "2:",
		"google":    "3:",
		"facebook":  "4:",
		"microsoft": "5:",
	}
	for _, item := range iutil.QueryAllUsers() {
		for k, v := range prefixes {
			if strings.HasPrefix(item.Snowflake, v) {
				sn := item.Snowflake[len(v):]
				util.Log("[db-upgrade]", item.Snowflake, "is now", sn, "as", k)
				DB.Build().Up("users", "snowflake", sn).Wh("id", item.IDS).Exe()
				DB.Build().Up("users", "provider", k).Wh("id", item.IDS).Exe()
			}
		}
	}
}
