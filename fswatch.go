package main

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/iutil"

	"github.com/fsnotify/fsnotify"
	"github.com/karrick/godirwalk"
	"github.com/nektro/go-util/sqlite"
	"github.com/nektro/go-util/util"
	etc "github.com/nektro/go.etc"
)

//
//

type WatchedFile struct {
	ID   int    `json:"id"`
	Path string `json:"path" sqlite:"text"`
	Name string `json:"name" sqlite:"text"`
	URL  string `json:"url"`
}

func scanFile(rows *sql.Rows) WatchedFile {
	var v WatchedFile
	rows.Scan(&v.ID, &v.Path, &v.Name)
	return v
}

//
//

var (
	watcher *fsnotify.Watcher
)

func initFsWatcher() {
	// creates a new file watcher
	watcher, _ = fsnotify.NewWatcher()
	etc.Database.CreateTableStruct("files", WatchedFile{})

	if err := godirwalk.Walk(idata.Config.Root, &godirwalk.Options{Callback: wWatchDir, Unsorted: true, FollowSymbolicLinks: true}); err != nil {
		util.LogError(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// util.Log("fsnotify", "event", event.Name, event.Op.String())
				r0 := strings.TrimPrefix(event.Name, idata.Config.Root)
				r1 := strings.Replace(r0, string(filepath.Separator), "/", -1)
				switch event.Op {
				case fsnotify.Rename, fsnotify.Remove:
					if sqlite.QueryHasRows(etc.Database.QueryPrepared(false, "select * from files where path = ?", r1)) {
						etc.Database.QueryPrepared(true, "delete from files where path = ?", r1)
					} else {
						r2 := r1 + "/"
						etc.Database.QueryPrepared(true, "delete from files where substr(path,1,length(?)) = ?", r2, r2)
					}
					util.Log("[file-index-del]", r1)
				case fsnotify.Create:
					f, _ := os.Stat(event.Name)
					if !f.IsDir() {
						n := f.Name()
						i := etc.Database.QueryNextID("files")
						etc.Database.QueryPrepared(true, "insert into files values (?, ?, ?)", i, r1, n)
						util.Log("[file-index-add]", r1)
					} else {
						if err := godirwalk.Walk(idata.Config.Root, &godirwalk.Options{Callback: wWatchDir, Unsorted: true, FollowSymbolicLinks: true}); err != nil {
							util.LogError(err)
						}
					}
				}
			case err := <-watcher.Errors:
				util.LogError("[fsnotify]", err)
			}
		}
	}()

	http.HandleFunc("/search", iutil.Mw(handleSearch))
	http.HandleFunc("/api/search", iutil.Mw(handleSearchAPI))
}

func wWatchDir(path string, de *godirwalk.Dirent) error {
	if de.IsDir() {
		return watcher.Add(path)
	}
	wAddFile(strings.TrimPrefix(path, idata.Config.Root), de.Name())
	return nil
}

func wAddFile(path string, name string) {
	pth := strings.Replace(path, string(filepath.Separator), "/", -1)
	if sqlite.QueryHasRows(etc.Database.QueryPrepared(false, "select * from files where path = ?", pth)) {
		return
	}
	id := etc.Database.QueryNextID("files")
	etc.Database.QueryPrepared(true, "insert into files values (?, ?, ?)", id, pth, name)
	util.Log("[file-index-add]", pth)
}
