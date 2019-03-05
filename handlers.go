package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/kataras/go-sessions"
	"github.com/valyala/fasthttp"
)

// handler for http://andesite/login
func handleOAuthLogin(ctx *fasthttp.RequestCtx) {
	sess := sessions.StartFasthttp(ctx)
	u := sess.Get("user")
	ctx.SetStatusCode(301)
	if u != nil {
		ctx.Response.Header.Add("Location", "./files/")
	} else {
		urlR, _ := url.Parse(oauth2Provider.authorizeURL)
		parameters := url.Values{}
		parameters.Add("client_id", oauth2AppID)
		parameters.Add("redirect_uri", fullHost(ctx)+httpBase+"callback")
		parameters.Add("response_type", "code")
		parameters.Add("scope", oauth2Provider.scope)
		parameters.Add("duration", "temporary")
		parameters.Add("state", "none")
		urlR.RawQuery = parameters.Encode()
		ctx.Response.Header.Add("Location", urlR.String())
	}
}

// handler for http://andesite/callback
func handleOAuthCallback(ctx *fasthttp.RequestCtx) {
	code := ctx.URI().QueryArgs().Peek("code")
	if len(code) == 0 {
		return
	}
	parameters := url.Values{}
	parameters.Add("client_id", oauth2AppID)
	parameters.Add("client_secret", oauth2AppSecret)
	parameters.Add("grant_type", "authorization_code")
	parameters.Add("code", string(code))
	parameters.Add("redirect_uri", fullHost(ctx)+httpBase+"callback")
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
	sess := sessions.StartFasthttp(ctx)
	sess.Set(accessToken, respJSON.AccessToken)
	ctx.SetStatusCode(301)
	ctx.Response.Header.Add("Location", "./token")
}

// handler for http://andesite/token
func handleOAuthToken(ctx *fasthttp.RequestCtx) {
	sess := sessions.StartFasthttp(ctx)
	val := sess.Get(accessToken)
	if val == nil {
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
	sess.Set("user", _id)
	sess.Set("name", _name)
	queryAssertUserName(_id, _name)

	ctx.SetStatusCode(301)
	ctx.Response.Header.Add("Location", "./files/")
}

// handler for http://andesite/test
func handleTest(ctx *fasthttp.RequestCtx) {
	// sessions test
	// increment number every refresh
	sess := sessions.StartFasthttp(ctx)
	i := sess.Get("int")
	if i == nil {
		i = 0
	}
	j := i.(int)
	sess.Set("int", j+1)
	fmt.Fprintf(ctx, strconv.Itoa(j))
}

// handler for http://andesite/files/*
func handleFileListing(ctx *fasthttp.RequestCtx) {
	if strings.Contains(string(ctx.Request.URI().Path()), "..") {
		return
	}

	sess := sessions.StartFasthttp(ctx)
	sessID := sess.Get("user")
	if sessID == nil {
		writeUserDenied(ctx, true, true)
		return
	}
	userID := sessID.(string)

	// get path
	// remove /files
	qpath := string(ctx.URI().Path()[6:])

	// disallow exploring dotfile folders
	if strings.Contains(qpath, "/.") {
		writeUserDenied(ctx, true, false)
		return
	}

	// valid path check
	stat, err := rootDir.Stat(qpath)
	if os.IsNotExist(err) {
		// 404
		writeUserDenied(ctx, true, false)
		return
	}

	// server file/folder
	if stat.IsDir() {
		ctx.Response.Header.Add("Content-Type", "text/html")

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
			writeUserDenied(ctx, true, false)
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

		sessName := sess.Get("name")
		writeHandlebarsFile(ctx, "/listing.hbs", map[string]interface{}{
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
			writeUserDenied(ctx, true, false)
			return
		}

		reader, _ := rootDir.ReadFile(qpath)
		data, _ := ioutil.ReadAll(reader)
		ctx.SetBody(data)
	}
}

// handler for http://andesite/admin
func handleAdmin(ctx *fasthttp.RequestCtx) {
	// get discord snowflake from session cookie
	sess := sessions.StartFasthttp(ctx)
	sessID := sess.Get("user")
	if sessID == nil {
		writeUserDenied(ctx, false, true)
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
		writeUserDenied(ctx, false, false)
		return
	}

	//
	accesses := queryAllAccess()
	sessName := sess.Get("name")
	writeHandlebarsFile(ctx, "/admin.hbs", map[string]interface{}{
		"user":     userID,
		"accesses": accesses,
		"base":     httpBase,
		"name":     oauth2Provider.namePrefix + sessName.(string),
	})
}

// handler for http://andesite/api/access/delete
func handleAccessDelete(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != http.MethodPost {
		writeAPIResponse(ctx, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := sessions.StartFasthttp(ctx)
	sessID := sess.Get("user")
	if sessID == nil {
		writeAPIResponse(ctx, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(ctx, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(ctx, false, "This action requires being a site administrator")
		return
	}
	//
	pa := ctx.PostArgs()
	if pa == nil {
		writeAPIResponse(ctx, false, "Error parsing form data")
		return
	}
	//
	aid := string(pa.Peek("id"))
	iid, err := strconv.ParseInt(aid, 10, 32)
	if err != nil {
		writeAPIResponse(ctx, false, "ID parameter must be an integer")
		return
	}
	//
	query(fmt.Sprintf("delete from access where id = '%d'", iid), true)
	writeAPIResponse(ctx, true, fmt.Sprintf("Removed access from %s.", string(pa.Peek("snowflake"))))
}

// handler for http://andesite/api/access/update
func handleAccessUpdate(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != http.MethodPost {
		writeAPIResponse(ctx, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := sessions.StartFasthttp(ctx)
	sessID := sess.Get("user")
	if sessID == nil {
		writeAPIResponse(ctx, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(ctx, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(ctx, false, "This action requires being a site administrator")
		return
	}
	//
	pa := ctx.PostArgs()
	if pa == nil {
		writeAPIResponse(ctx, false, "Error parsing form data")
		return
	}
	//
	aid := pa.Peek("id")
	iid, err := strconv.ParseInt(string(aid), 10, 32)
	if err != nil {
		writeAPIResponse(ctx, false, "ID parameter must be an integer")
		return
	}
	//
	queryPrepared("update access set path = ? where id = ?", true, string(pa.Peek("path")), iid)
	writeAPIResponse(ctx, true, fmt.Sprintf("Updated access for %s.", string(pa.Peek("snowflake"))))
}

// handler for http://andesite/api/access/create
func handleAccessCreate(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != http.MethodPost {
		writeAPIResponse(ctx, false, "This action requires using HTTP POST")
		return
	}
	//
	sess := sessions.StartFasthttp(ctx)
	sessID := sess.Get("user")
	if sessID == nil {
		writeAPIResponse(ctx, false, "This action requires being logged in")
		return
	}
	userID := sessID.(string)
	//
	user, ok := queryUserBySnowflake(userID)
	if !ok {
		writeAPIResponse(ctx, false, "This action requires being a member of this server")
		return
	}
	if !user.admin {
		writeAPIResponse(ctx, false, "This action requires being a site administrator")
		return
	}
	//
	pa := ctx.PostArgs()
	if pa == nil {
		writeAPIResponse(ctx, false, "Error parsing form data")
		return
	}
	//
	aid := queryLastID("access") + 1
	asn := string(pa.Peek("snowflake"))
	apt := string(pa.Peek("path"))
	//
	u, ok := queryUserBySnowflake(asn)
	aud := -1
	if ok {
		aud = u.id
	} else {
		aud = queryLastID("users") + 1
		queryPrepared("insert into users values (?, ?, ?, ?)", true, aud, oauth2Provider.dbPrefix+asn, 0, "")
	}
	//
	queryPrepared("insert into access values (?, ?, ?)", true, aid, aud, apt)
	writeAPIResponse(ctx, true, fmt.Sprintf("Created access for %s.", asn))
}
