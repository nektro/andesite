package iutil

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	discord "github.com/nektro/go.discord"
	etc "github.com/nektro/go.etc"
	oauth2 "github.com/nektro/go.oauth2"

	"github.com/nektro/andesite/internal/idata"
	"github.com/nektro/andesite/internal/itypes"

	. "github.com/nektro/go-util/alias"
)

var (
	Mw = ChainMiddleware(MwAddAttribution)
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
		me += F("%s%s (%s)", oauth2.ProviderIDMap[provider].NamePrefix, sessName.(string), sessID)
	}

	message := ""
	if fileOrAdmin {
		if showLogin {
			message = "You " + me + " do not have access to this resource."
		} else {
			message = "Unable to find the requested resource for you " + me + "."
		}
	} else {
		message = "Admin priviledge required. Access denied."
	}

	linkmsg := ""
	if showLogin {
		linkmsg = "Please <a href='" + idata.Config.HTTPBase + "login'>Log In</a>."
		w.WriteHeader(http.StatusForbidden)
		WriteResponse(r, w, "Forbidden", message, linkmsg)
	} else {
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
	WriteResponse(r, w, titlemsg, message, "Return to <a href='"+idata.Config.HTTPBase+"admin'>the dashboard</a>.")
}

func BoolToString(x bool) string {
	if x {
		return "1"
	}
	return "0"
}

func WriteResponse(r *http.Request, w http.ResponseWriter, title string, message string, link string) {
	etc.WriteHandlebarsFile(r, w, "/response.hbs", map[string]interface{}{
		"title":   title,
		"message": message,
		"link":    link,
		"base":    idata.Config.HTTPBase,
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

func ApiBootstrapRequireLogin(r *http.Request, w http.ResponseWriter, method string, requireAdmin bool) (*sessions.Session, *itypes.UserRow, error) {
	if r.Method != method {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Add("Allow", "HEAD, "+method)
		WriteAPIResponse(r, w, false, "This action requires using HTTP "+method)
		return nil, nil, E("")
	}

	sess := etc.GetSession(r)
	sessID := sess.Values["user"]

	if sessID == nil {
		pk := ""

		if len(pk) == 0 {
			ua := r.Header.Get("user-agent")
			if strings.HasPrefix(ua, "AndesiteUser/") {
				pk = strings.Split(ua, "/")[1]
			}
		}
		if len(pk) == 0 {
			pk = r.Header.Get("x-passkey")
		}
		if len(pk) == 0 {
			WriteUserDenied(r, w, true, true)
			return nil, nil, E("not logged in and no passkey found")
		}
		kq := etc.Database.Build().Se("*").Fr("users").Wh("passkey", pk).Exe()
		if !kq.Next() {
			WriteUserDenied(r, w, true, true)
			return nil, nil, E("invalid passkey")
		}
		sessID = ScanUser(kq).Snowflake
		kq.Close()
	}

	userID := sessID.(string)
	user, ok := QueryUserBySnowflake(userID)

	if !ok {
		WriteResponse(r, w, "Access Denied", "This action requires being a member of this server. ("+userID+")", "")
		return nil, nil, E("")
	}
	if requireAdmin && !user.Admin {
		WriteAPIResponse(r, w, false, "This action requires being a site administrator. ("+userID+")")
		return nil, nil, E("")
	}

	err := r.ParseForm()
	if err != nil {
		WriteAPIResponse(r, w, false, "Error parsing form data")
		return nil, nil, E("")
	}

	return sess, user, nil
}

// @from https://gist.github.com/gbbr/fa652db0bab132976620bcb7809fd89a
func ChainMiddleware(mw ...itypes.Middleware) itypes.Middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i](last)
			}
			last(w, r)
		}
	}
}

func MwAddAttribution(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server", "nektro/andesite")
		next.ServeHTTP(w, r)
	}
}

func FindFirstNonEmpty(values ...string) string {
	for _, item := range values {
		if len(item) > 0 {
			return item
		}
	}
	return ""
}

func FindFirstNonZero(values ...int) int {
	for _, item := range values {
		if item != 0 {
			return item
		}
	}
	return 0
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
	req, _ := http.NewRequest(http.MethodGet, idata.DiscordAPI+endpoint, strings.NewReader(body.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bot "+idata.Config.GetDiscordClient().Extra2)
	req.Header.Set("Accept", "application/json")
	res, _ := http.DefaultClient.Do(req)
	bys, _ := ioutil.ReadAll(res.Body)
	return bys
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
