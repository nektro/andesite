package main

import (
	"os"
	"strings"
	"time"
)

//
type HttpFileInfo struct {
	parentDir string
	name      string
	isDir     bool
}

//
func (hf HttpFileInfo) Name() string {
	if hf.isDir {
		return strings.TrimSuffix(hf.name, "/")
	} else {
		return hf.name
	}
}

//
func (hf HttpFileInfo) Size() int64 {
	return 0
}

//
func (hf HttpFileInfo) Mode() os.FileMode {
	if hf.isDir {
		return os.ModePerm
	} else {
		return os.ModeDir
	}
}

//
func (hf HttpFileInfo) ModTime() time.Time {
	return time.Now()
}

//
func (hf HttpFileInfo) IsDir() bool {
	return hf.isDir
}

//
func (hf HttpFileInfo) Sys() interface{} {
	return nil
}
