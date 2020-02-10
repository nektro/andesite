package fs

import (
	"io"
	"os"
)

type FileHandle interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Fs interface {
	Stat(name string) (os.FileInfo, error)
	Open(name string) (FileHandle, error)
}

type localStorage struct{}

var LocalStorage = &localStorage{}

func (l *localStorage) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (l *localStorage) Open(name string) (FileHandle, error) {
	return os.Open(name)
}
