package main

import (
	"net/http"
)

//
type OAuth2CallBackResponse struct {
	AccessToken string `json:"access_token"`
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
	name      string
}

//
type ShareRow struct {
	id   int
	hash string
	path string
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
