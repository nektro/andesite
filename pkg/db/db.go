package db

import (
	"strings"

	"github.com/nektro/andesite/pkg/itypes"
	"github.com/nektro/andesite/pkg/iutil"

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
