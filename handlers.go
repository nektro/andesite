package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/nektro/go-util/util"
)

// handler for http://andesite/login
func handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	_, ok := sess.Values["user"]
	if ok {
		w.Header().Add("Location", "./files/")
	} else {
		urlR, _ := url.Parse(oauth2Provider.idp.AuthorizeURL)
		parameters := url.Values{}
		parameters.Add("client_id", oauth2AppID)
		parameters.Add("redirect_uri", fullHost(r)+httpBase+"callback")
		parameters.Add("response_type", "code")
		parameters.Add("scope", oauth2Provider.idp.Scope)
		parameters.Add("duration", "temporary")
		parameters.Add("state", "none")
		urlR.RawQuery = parameters.Encode()
		w.Header().Add("Location", urlR.String())
	}
	w.WriteHeader(http.StatusMovedPermanently)
}

// handler for http://andesite/callback
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		return
	}

	parameters := url.Values{}
	parameters.Add("client_id", oauth2AppID)
	parameters.Add("client_secret", oauth2AppSecret)
	parameters.Add("grant_type", "authorization_code")
	parameters.Add("code", string(code))
	parameters.Add("redirect_uri", fullHost(r)+httpBase+"callback")
	parameters.Add("state", "none")

	urlR, _ := url.Parse(oauth2Provider.idp.TokenURL)
	req, _ := http.NewRequest("POST", urlR.String(), strings.NewReader(parameters.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(oauth2AppID+":"+oauth2AppSecret)))
	req.Header.Set("Accept", "application/json")


	body := doHttpRequest(req)
	var respJSON OAuth2CallBackResponse
	json.Unmarshal(body, &respJSON)
	sess := getSession(r)
	sess.Values[accessToken] = respJSON.AccessToken
	sess.Save(r, w)
	w.Header().Add("Location", "./token")
	w.WriteHeader(http.StatusMovedPermanently)
}

// handler for http://andesite/token
func handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	val, ok := sess.Values[accessToken]
	if !ok {
		return
	}

	urlR, _ := url.Parse(oauth2Provider.idp.MeURL)
	req, _ := http.NewRequest("GET", urlR.String(), strings.NewReader(""))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Authorization", "Bearer "+val.(string))


	body := doHttpRequest(req)
	var respMe map[string]interface{}
	json.Unmarshal(body, &respMe)
	_id := fixID(respMe["id"])
	_name := respMe[oauth2Provider.idp.NameProp].(string)
	sess.Values["user"] = _id
	sess.Values["name"] = _name
	sess.Save(r, w)
	queryAssertUserName(_id, _name)

	w.Header().Add("Location", "./files/")
	w.WriteHeader(http.StatusMovedPermanently)
}

// handler for http://andesite/test
func handleTest(w http.ResponseWriter, r *http.Request) {
	// sessions test
	// increment number every refresh
	sess := getSession(r)
	i := sess.Values["int"]
	if i == nil {
		i = 0
	}
	j := i.(int)
	sess.Values["int"] = j + 1
	sess.Save(r, w)
	fmt.Fprintf(w, strconv.Itoa(j))
}

func handleDirectoryListing(getAccess func(http.ResponseWriter, *http.Request) (string, []string, string, string, bool, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		qpath, uAccess, uID, uName, isAdmin, err := getAccess(w, r)

		// if getAccess errored, response has already been written
		if err != nil {
			return
		}

		// disallow path tricks
		if strings.Contains(string(r.URL.Path), "..") {
			return
		}

		// disallow exploring dotfile folders
		if strings.Contains(qpath, "/.") {
			writeUserDenied(r, w, true, false)
			return
		}

		// valid path check
		stat, err := rootDir.Stat(qpath)
		if os.IsNotExist(err) {
			// 404
			writeUserDenied(r, w, true, false)
			return
		}

		// server file/folder
		if stat.IsDir() {
			w.Header().Add("Content-Type", "text/html")

			// get list of all files
			files, _ := rootDir.ReadDir(qpath)

			// hide dot files
			files = filter(files, func(x os.FileInfo) bool {
				return !strings.HasPrefix(x.Name(), ".")
			})

			// amount of files in the directory
			l1 := len(files)

			// access check
			files = filter(files, func(x os.FileInfo) bool {
				ok := false
				fpath := qpath + x.Name()
				for _, item := range uAccess {
					if strings.HasPrefix(item, fpath) || strings.HasPrefix(qpath, item) {
						ok = true
					}
				}
				return ok
			})

			// amount of files given access to
			l2 := len(files)

			if l1 > 0 && l2 == 0 {
				writeUserDenied(r, w, true, false)
				return
			}

			data := make([]map[string]string, len(files))
			gi := 0
			for i := 0; i < len(files); i++ {
				name := files[i].Name()
				a := ""
				if files[i].IsDir() || files[i].Mode()&os.ModeSymlink != 0 {
					a = name + "/"
				} else {
					a = name
				}
				data[gi] = map[string]string{
					"name": a,
					"size": byteCountIEC(files[i].Size()),
					"mod":  files[i].ModTime().UTC().String()[:19],
				}
				gi++
			}

			writeHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
				"user":  uID,
				"path":  qpath,
				"files": data,
				"admin": isAdmin,
				"base":  httpBase,
				"name":  oauth2Provider.idp.NamePrefix + uName,
			})
		} else {
			// access check
			can := false
			for _, item := range uAccess {
				if strings.HasPrefix(qpath, item) {
					can = true
				}
			}
			if can == false {
				writeUserDenied(r, w, true, false)
				return
			}

			w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(qpath)))
			file, _ := rootDir.ReadFile(qpath)
			info, _ := rootDir.Stat(qpath)
			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
		}
	}
}

// handler for http://andesite/files/*
func handleFileListing(w http.ResponseWriter, r *http.Request) (string, []string, string, string, bool, error) {
	_, user, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if errr != nil {
		return "", []string{}, "", "", false, errors.New("")
	}

	// get path
	// remove /files
	qpath := string(r.URL.Path[6:])

	userUser, _ := queryUserBySnowflake(user.snowflake)
	userAccess := queryAccess(user.snowflake)

	return qpath, userAccess, user.snowflake, user.name, userUser.admin, nil
}

// handler for http://andesite/admin
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	_, user, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, true)
	if errr != nil {
		return
	}

	//
	accesses := queryAllAccess()
	shares := queryAllShares()
	writeHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"user":     user.snowflake,
		"accesses": accesses,
		"base":     httpBase,
		"name":     oauth2Provider.idp.NamePrefix + user.name,
		"shares":   shares,
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "snowflake") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := r.PostForm.Get("id")
	iid, err := strconv.ParseInt(aid, 10, 32)
	if err != nil {
		writeAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	database.Query(true, fmt.Sprintf("delete from access where id = '%d'", iid))
	writeAPIResponse(r, w, true, fmt.Sprintf("Removed access from %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	aid := r.PostForm.Get("id")
	iid, err := strconv.ParseInt(string(aid), 10, 32)
	if err != nil {
		writeAPIResponse(r, w, false, "ID parameter must be an integer")
		return
	}
	//
	if !containsAll(r.PostForm, "snowflake", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	queryDoUpdate("access", "path", r.PostForm.Get("path"), "id", strconv.FormatInt(iid, 10))
	writeAPIResponse(r, w, true, fmt.Sprintf("Updated access for %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "snowflake", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := database.QueryNextID("access")
	asn := r.PostForm.Get("snowflake")
	apt := r.PostForm.Get("path")
	//
	u, ok := queryUserBySnowflake(asn)
	aud := -1
	if ok {
		aud = u.id
	} else {
		aud = database.QueryNextID("users")
		queryDoAddUser(aud, asn, false, "")
	}
	//
	database.QueryPrepared(true, "insert into access values (?, ?, ?)", aid, aud, apt)
	writeAPIResponse(r, w, true, fmt.Sprintf("Created access for %s.", asn))
}

func handleShareCreate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	aid := database.QueryNextID("shares")
	ahs1 := md5.Sum([]byte(fmt.Sprintf("astheno.andesite.share.%s.%s", strconv.FormatInt(int64(aid), 10), util.GetIsoDateTime())))
	ahs2 := hex.EncodeToString(ahs1[:])
	fpath := r.PostForm.Get("path")
	//
	database.QueryPrepared(true, "insert into shares values (?, ?, ?)", aid, ahs2, fpath)
	writeAPIResponse(r, w, true, fmt.Sprintf("Created share with code %s for folder %s.", ahs2, fpath))
}

func handleShareListing(w http.ResponseWriter, r *http.Request) (string, []string, string, string, bool, error) {
	u := r.URL.Path[6:]
	if len(u) == 0 {
		w.Header().Add("Location", "../")
		w.WriteHeader(http.StatusMovedPermanently)
	}
	if match, _ := regexp.MatchString("^[0-9a-f]{32}/.*", u); !match {
		writeResponse(r, w, "Invalid Share Link", "Invalid format for share code.", "")
		return "", []string{}, "", "", false, errors.New("")
	}

	h := u[:32]
	s := queryAccessByShare(h)
	if len(s) == 0 {
		writeResponse(r, w, "Not Found", "Public share code not found.", "")
		return "", []string{}, "", "", false, errors.New("")
	}

	return u[32:], s, h, "", false, nil
}

func handleShareUpdate(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "hash", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	aph := r.PostForm.Get("path")
	// //
	queryDoUpdate("shares", "path", aph, "hash", ahs)
	writeAPIResponse(r, w, true, "Successfully updated share path.")
}

func handleShareDelete(w http.ResponseWriter, r *http.Request) {
	_, _, errr := apiBootstrapRequireLogin(r, w, http.MethodPost, true)
	if errr != nil {
		return
	}
	//
	if !containsAll(r.PostForm, "id", "hash", "path") {
		writeAPIResponse(r, w, false, "Missing POST values")
		return
	}
	//
	ahs := r.PostForm.Get("hash")
	//
	database.QueryPrepared(true, "delete from shares where hash = ?", ahs)
	writeAPIResponse(r, w, true, "Successfully deleted share link.")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _, errr := apiBootstrapRequireLogin(r, w, http.MethodGet, false)
	if errr != nil {
		return
	}
	//
	sess.Options.MaxAge = -1
	sess.Save(r, w)
	writeResponse(r, w, "Success", "Successfully logged out.", "")
}
