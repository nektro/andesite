package main

import (
	"net/http"
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
