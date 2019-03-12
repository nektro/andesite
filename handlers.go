package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

// handler for http://andesite/login
func handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	_, ok := sess.Values["user"]
	if ok {
		w.Header().Add("Location", "./files/")
	} else {
		urlR, _ := url.Parse(oauth2Provider.authorizeURL)
		parameters := url.Values{}
		parameters.Add("client_id", oauth2AppID)
		parameters.Add("redirect_uri", fullHost(r)+httpBase+"callback")
		parameters.Add("response_type", "code")
		parameters.Add("scope", oauth2Provider.scope)
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
	urlR, _ := url.Parse(oauth2Provider.tokenURL)
	req, _ := http.NewRequest("POST", urlR.String(), strings.NewReader(parameters.Encode()))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(oauth2AppID+":"+oauth2AppSecret)))
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
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
	urlR, _ := url.Parse(oauth2Provider.meURL)
	req, _ := http.NewRequest("GET", urlR.String(), strings.NewReader(""))
	req.Header.Set("User-Agent", "nektro/andesite")
	req.Header.Set("Authorization", "Bearer "+val.(string))
	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var respMe map[string]interface{}
	json.Unmarshal(body, &respMe)
	_id := fixID(respMe["id"])
	_name := respMe[oauth2Provider.nameProp].(string)
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

// handler for http://andesite/files/*
func handleFileListing(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(string(r.URL.Path), "..") {
		return
	}

	sess := getSession(r)
	sessID := sess.Values["user"]
	if sessID == nil {
		writeUserDenied(r, w, true, true)
		return
	}
	userID := sessID.(string)

	// get path
	// remove /files
	qpath := string(r.URL.Path[6:])

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
		acc := queryAccess(userID)
		files = filter(files, func(x os.FileInfo) bool {
			ok := false
			fpath := qpath + x.Name()
			for _, item := range acc {
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
			b := byteCountIEC(files[i].Size())
			c := files[i].ModTime().UTC().String()[:19]
			data[gi] = map[string]string{
				"name": a,
				"size": b,
				"mod":  c,
			}
			gi++
		}

		useruser, ok := queryUserBySnowflake(userID)
		admin := false
		if ok {
			admin = useruser.admin
		}

		sessName := sess.Values["name"]
		writeHandlebarsFile(r, w, "/listing.hbs", map[string]interface{}{
			"user":  userID,
			"path":  qpath,
			"files": data,
			"admin": admin,
			"base":  httpBase,
			"name":  oauth2Provider.namePrefix + sessName.(string),
		})
	} else {
		// access check
		can := false
		for _, item := range queryAccess(userID) {
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

// handler for http://andesite/admin
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	// get discord snowflake from session cookie
	sess := getSession(r)
	sessID := sess.Values["user"]
	if sessID == nil {
		writeUserDenied(r, w, false, true)
		return
	}
	userID := sessID.(string)

	// only allow admins
	useruser, ok := queryUserBySnowflake(userID)
	admin := false
	if ok {
		admin = useruser.admin
	}
	if !admin {
		writeUserDenied(r, w, false, false)
		return
	}

	//
	accesses := queryAllAccess()
	sessName := sess.Values["name"]
	writeHandlebarsFile(r, w, "/admin.hbs", map[string]interface{}{
		"user":     userID,
		"accesses": accesses,
		"base":     httpBase,
		"name":     oauth2Provider.namePrefix + sessName.(string),
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIResponse(r, w, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := getSession(r)
	sessID := sess.Values["user"]
	if sessID == nil {
		writeAPIResponse(r, w, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(r, w, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator")
		return
	}
	//
	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
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
	query(fmt.Sprintf("delete from access where id = '%d'", iid), true)
	writeAPIResponse(r, w, true, fmt.Sprintf("Removed access from %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIResponse(r, w, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := getSession(r)
	sessID := sess.Values["user"]
	if sessID == nil {
		writeAPIResponse(r, w, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(r, w, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator")
		return
	}
	//
	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
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
	queryDoUpdate("access", "path", r.PostForm.Get("path"), "id", strconv.FormatInt(iid, 10))
	writeAPIResponse(r, w, true, fmt.Sprintf("Updated access for %s.", r.PostForm.Get("snowflake")))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIResponse(r, w, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := getSession(r)
	sessID := sess.Values["user"]
	if sessID == nil {
		writeAPIResponse(r, w, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(r, w, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(r, w, false, "This action requires being a site administrator")
		return
	}
	//
	err := r.ParseForm()
	if err != nil {
		writeAPIResponse(r, w, false, "Error parsing form data")
		return
	}
	//
	aid := queryLastID("access") + 1
	asn := r.PostForm.Get("snowflake")
	apt := r.PostForm.Get("path")
	//
	u, ok := queryUserBySnowflake(asn)
	aud := -1
	if ok {
		aud = u.id
	} else {
		aud = queryLastID("users") + 1
		queryDoAddUser(aud, oauth2Provider.dbPrefix+asn, false, "")
	}
	//
	queryPrepared("insert into access values (?, ?, ?)", true, aid, aud, apt)
	writeAPIResponse(r, w, true, fmt.Sprintf("Created access for %s.", asn))
}
