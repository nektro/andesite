package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nektro/go-util/util"
	discord "github.com/nektro/go.discord"
	etc "github.com/nektro/go.etc"

	. "github.com/nektro/go-util/alias"

	"github.com/nektro/andesite/config"
)

func Filter(stack []os.FileInfo, cb func(os.FileInfo) bool) []os.FileInfo {
	result := []os.FileInfo{}
	for _, item := range stack {
		if cb(item) {
			result = append(result, item)
		}
	}
	return result
}

func WriteUserDenied(r *http.Request, w http.ResponseWriter, fileOrAdmin bool, showLogin bool) {
	me := ""
	sess := etc.GetSession(r)
	sessName := sess.Values["name"]
	if sessName != nil {
		sessID := sess.Values["user"].(string)
		provider := sess.Values["provider"].(string)
		me += F(" %s@%s (%s)", sessName.(string), provider, sessID)
	}

	message := ""
	if fileOrAdmin {
		if showLogin {
			message = "You " + me + " do not have access to this resource."
		} else {
			message = "Unable to find the requested resource for you" + me + "."
		}
	} else {
		message = "Admin priviledge required. Access denied."
	}

	linkmsg := ""
	if showLogin {
		linkmsg = "Please <a href='" + config.Config.HTTPBase + "login'>Log In</a>."
		w.WriteHeader(http.StatusForbidden)
		WriteResponse(r, w, "Forbidden", message, linkmsg)
	} else {
		linkmsg = "<a href='" + config.Config.HTTPBase + "logout'>Logout</a>."
		w.WriteHeader(http.StatusForbidden)
		WriteResponse(r, w, "Not Found", message, linkmsg)
	}
}

func WriteAPIResponse(r *http.Request, w http.ResponseWriter, good bool, message string) {
	if !good {
		w.WriteHeader(http.StatusForbidden)
	}
	titlemsg := ""
	if good {
		titlemsg = "Update Successful"
	} else {
		titlemsg = "Update Failed"
	}
	WriteResponse(r, w, titlemsg, message, "Return to <a href='"+config.Config.HTTPBase+"admin'>the dashboard</a>.")
}

func BoolToString(x bool) string {
	if x {
		return "1"
	}
	return "0"
}

func WriteResponse(r *http.Request, w http.ResponseWriter, title string, message string, link string) {
	etc.WriteHandlebarsFile(r, w, "/response.hbs", map[string]interface{}{
		"version": config.Version,
		"title":   title,
		"message": message,
		"link":    link,
		"base":    config.Config.HTTPBase,
	})
}

func WriteLinkResponse(r *http.Request, w http.ResponseWriter, title string, message string, linkText string, href string) {
	WriteResponse(r, w, title, message, "<a href=\""+href+"\">"+linkText+"</a>")
}

func ContainsAll(mp url.Values, keys ...string) bool {
	for _, item := range keys {
		if _, ok := mp[item]; !ok {
			return false
		}
	}
	return true
}

func WriteJSON(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("content-type", "application/json")
	bytes, _ := json.Marshal(data)
	w.Write(bytes)
}

func GenerateNewUserPasskey(snowflake string) string {
	hash1 := md5.Sum([]byte(F("astheno.andesite.passkey.%s.%s", snowflake, T())))
	hash2 := hex.EncodeToString(hash1[:])
	return hash2[0:10]
}

func MakeDiscordRequest(endpoint string, body url.Values) []byte {
	req, _ := http.NewRequest(http.MethodGet, config.DiscordAPI+endpoint, strings.NewReader(body.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bot "+config.Config.GetDiscordClient().Extra2)
	req.Header.Set("Accept", "application/json")
	return util.DoHttpRequest(req)
}

func FetchDiscordRole(guild string, role string) discord.GuildRole {
	bys := MakeDiscordRequest("/guilds/"+guild+"/roles", url.Values{})
	roles := []discord.GuildRole{}
	json.Unmarshal(bys, &roles)
	for i, item := range roles {
		if item.ID == role {
			return roles[i]
		}
	}
	return discord.GuildRole{}
}

type DiscordGuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func FetchDiscordGuild(guild string) DiscordGuild {
	bys := MakeDiscordRequest("/guilds/"+guild, url.Values{})
	var dg DiscordGuild
	json.Unmarshal(bys, &dg)
	return dg
}

func FilterStr(stack []string, cb func(string) bool) []string {
	result := []string{}
	for _, item := range stack {
		if cb(item) {
			result = append(result, item)
		}
	}
	return result
}

func MapStr(stack []string, cb func(string) string) []string {
	result := []string{}
	for _, item := range stack {
		result = append(result, cb(item))
	}
	return result
}
