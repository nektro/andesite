package db

import (
	"net/http"
	"strings"

	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"
	etc "github.com/nektro/go.etc"
)

var (
	DB dbstorage.Database
	FS dbstorage.Database
)

const (
	ctUser              = "users"
	ctUserAccess        = "access"
	ctShare             = "shares"
	ctDiscordRoleAccess = "shares_discord_role"
	ctFile              = "files"
)

var (
	searchCache = map[string]bool{}
)

func Init() {
	DB = etc.Database
	DB.CreateTableStruct(ctUser, User{})
	DB.CreateTableStruct(ctUserAccess, UserAccess{})
	DB.CreateTableStruct(ctShare, Share{})
	DB.CreateTableStruct(ctDiscordRoleAccess, DiscordRoleAccess{})

	FS = dbstorage.ConnectSqlite(etc.DataRoot() + "/files.db")
	FS.CreateTableStruct(ctFile, File{})
}

func Upgrade() {
	prefixes := map[string]string{
		"reddit":    "1:",
		"github":    "2:",
		"google":    "3:",
		"facebook":  "4:",
		"microsoft": "5:",
	}
	for _, item := range (User{}.All()) {
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

func SaveOAuth2InfoCb(w http.ResponseWriter, r *http.Request, provider string, id string, name string, resp map[string]interface{}) {
	sess := etc.GetSession(r)
	sess.Values["provider"] = provider
	sess.Values["user"] = id
	sess.Values["name"] = name
	sess.Values[provider+"_access_token"] = resp["access_token"]
	sess.Values[provider+"_expires_in"] = resp["expires_in"]
	sess.Values[provider+"_refresh_token"] = resp["refresh_token"]
	sess.Save(r, w)
	QueryAssertUserName(provider, id, name)
	util.Log("[user-login]", provider, id, name)
}

func FolderSize(p string) (size int64, count int64) {
	rows := FS.Build().Se("sum(size), count(*)").Fr(ctFile).WR("path", "like", "?||'%'", true, p).Exe()
	defer rows.Close()
	rows.Next()
	rows.Scan(&size, &count)
	return
}

func CanSearch(p string) bool {
	b, ok := searchCache[p]
	if ok {
		return b
	}
	_, count := FolderSize(p)
	b = count > 0
	searchCache[p] = b
	return b
}
