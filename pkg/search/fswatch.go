package search

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/pkg/idata"

	"github.com/fsnotify/fsnotify"
	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"
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

func Init() {
	// creates a new file watcher
	watcher, _ = fsnotify.NewWatcher()
	etc.Database.CreateTableStruct("files", WatchedFile{})

	if err := filepath.Walk(idata.Config.Root, WatchDir); err != nil {
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
					if dbstorage.QueryHasRows(etc.Database.QueryPrepared(false, "select * from files where path = ?", r1)) {
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
						if err := filepath.Walk(event.Name, WatchDir); err != nil {
							util.LogError(err)
						}
					}
				}
			case err := <-watcher.Errors:
				util.LogError("[fsnotify]", err)
			}
		}
	}()

	http.HandleFunc("/search", HandleSearch)
	http.HandleFunc("/api/search", HandleSearchAPI)
}

func Close() {
	util.Log("Closing filesystem watcher")
	watcher.Close()
}

func WatchDir(path string, fi os.FileInfo, err error) error {
	if fi.IsDir() {
		return watcher.Add(path)
	}
	AddFile(strings.TrimPrefix(path, idata.Config.Root), fi.Name())
	return nil
}

func AddFile(path string, name string) {
	pth := strings.Replace(path, string(filepath.Separator), "/", -1)
	if dbstorage.QueryHasRows(etc.Database.QueryPrepared(false, "select * from files where path = ?", pth)) {
		return
	}
	id := etc.Database.QueryNextID("files")
	etc.Database.QueryPrepared(true, "insert into files values (?, ?, ?)", id, pth, name)
	util.Log("[file-index-add]", pth)
}
