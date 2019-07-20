package main

import (
	"io"
	"io/ioutil"
	"os"
)

//
type RootDir interface {
	ReadFile(string) (io.ReadSeeker, error)
	ReadDir(string) ([]os.FileInfo, error)
	Stat(string) (os.FileInfo, error)
	Base() string
}

//
type FsRoot struct {
	base string
}

//
func (rd FsRoot) ReadFile(fpath string) (io.ReadSeeker, error) {
	return os.Open(rd.base + fpath)
}

//
func (rd FsRoot) ReadDir(fpath string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(rd.base + fpath)
}

//
func (rd FsRoot) Stat(fpath string) (os.FileInfo, error) {
	return os.Stat(rd.base + fpath)
}

//
func (rd FsRoot) Base() string {
	return rd.base
}
