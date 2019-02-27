package main

import (
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/valyala/fasthttp"
)

type FusingFileSystem struct {
	fs []http.FileSystem
}

func (ffs FusingFileSystem) Open(name string) (http.File, error) {
	var errr error
	for _, item := range ffs.fs {
		file, err := item.Open(name)
		if file != nil {
			return file, nil
		}
		errr = err
	}
	return nil, errr
}

func (ffs FusingFileSystem) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	fpath := string(ctx.Request.RequestURI())
	if strings.HasSuffix(fpath, "/") {
		fpath += "index.html"
	}
	// fmt.Println("GET: ", fpath)
	file, err := ffs.Open(fpath)
	if err == nil {
		bytes, _ := ioutil.ReadAll(file)
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetContentType(mime.TypeByExtension(path.Ext(fpath)))
		ctx.SetBody(bytes)
	} else {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	}
}
