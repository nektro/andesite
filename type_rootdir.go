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

// //
// type HttpRoot struct {
// 	base string
// }

// //
// func (rd HttpRoot) ReadFile(fpath string) (io.ReadSeeker, error) {
// 	req, _ := http.NewRequest(http.MethodGet, rd.base+fpath, strings.NewReader(""))
// 	client := &http.Client{}
// 	resp, _ := client.Do(req)
// 	return resp.Body, nil
// }

// //
// func (rd HttpRoot) ReadDir(fpath string) ([]os.FileInfo, error) {
// 	req, _ := http.NewRequest(http.MethodGet, rd.base+fpath, strings.NewReader(""))
// 	client := &http.Client{}
// 	resp, _ := client.Do(req)
// 	defer resp.Body.Close()
// 	if resp.StatusCode != 200 {
// 		return nil, errors.New(resp.Status)
// 	}
// 	body, _ := ioutil.ReadAll(resp.Body)
// 	z := html.NewTokenizer(strings.NewReader(string(body)))
// 	result := []os.FileInfo{}

// 	for {
// 		tt := z.Next()
// 		if tt == html.ErrorToken {
// 			break
// 		}
// 		tk := z.Token()
// 		if tt != html.StartTagToken {
// 			continue
// 		}
// 		if len(tk.Attr) != 1 {
// 			continue
// 		}
// 		if tk.Attr[0].Key != "href" {
// 			continue
// 		}
// 		name, _ := url.PathUnescape(tk.Attr[0].Val)
// 		result = append(result, HttpFileInfo{
// 			rd.base + fpath,
// 			name,
// 			strings.HasSuffix(name, "/"),
// 		})
// 	}

// 	return result, nil
// }

// //
// func (rd HttpRoot) Stat(fpath string) (os.FileInfo, error) {
// 	req, _ := http.NewRequest(http.MethodHead, rd.base+fpath, strings.NewReader(""))
// 	client := &http.Client{}
// 	resp, _ := client.Do(req)
// 	if resp.StatusCode != 200 {
// 		return nil, os.ErrNotExist
// 	}
// 	hfi := HttpFileInfo{
// 		rd.base,
// 		fpath,
// 		strings.HasSuffix(fpath, "/"),
// 	}
// 	return hfi, nil
// }

// //
// func (rd HttpRoot) Base() string {
// 	return rd.base
// }
