package main

import (
	"net/http"
)

//
type Config struct {
	Auth    string       `json:"auth"`
	Discord oauth2Client `json:"discord"`
}

//
type oauth2Client struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

//
type OAuth2CallBackResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

//
type DiscordAPIMeResponse struct {
	Username      string `json:"username"`
	Locale        string `json:"locale"`
	PremiumType   int    `json:"premium_type"`
	Multifactor   bool   `json:"mfa_enabled"`
	Flags         int    `json:"flags"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
	ID            string `json:"id"`
}

//
type PragmaTableInfoRow struct {
	cid          int
	name         string
	rowType      string
	notnull      bool
	defaultValue interface{}
	pk           int
}

//
type UserAccessRow struct {
	id   int
	user int
	path string
}

//
type UserRow struct {
	id        int
	snowflake string
	admin     bool
}

// Middleware provides a convenient mechanism for augmenting HTTP requests
// entering the application. It returns a new handler which may perform various
// operations and should finish by calling the next HTTP handler.
//
// @from https://gist.github.com/gbbr/dc731df098276f1a135b343bf5f2534a
type Middleware func(next http.HandlerFunc) http.HandlerFunc

//
type RootDirType string

//
const (
	RootTypeDir  RootDirType = "dir"
	RootTypeHttp             = "http"
)
