package fsdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/itypes"

	"github.com/karrick/godirwalk"
	"github.com/nektro/go-util/util"
)

const (
	cTbl = "files"
)

func Init(mp map[string]string, rt string) {
	bd, ok := mp[rt]
	if !ok {
		return
	}
	util.Log("fsdb:", rt+":", "scan begin...")
	godirwalk.Walk(bd, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			fpathS, _ := filepath.Abs(osPathname)
			if strings.Contains(fpathS, "/.") {
				return nil
			}
			upathS := "/" + rt + strings.TrimPrefix(fpathS, bd)
			s, _ := os.Stat(osPathname)
			if s.IsDir() {
				return nil
			}
			insertFile(&itypes.File{
				0,
				rt,
				upathS,
				s.Size(), "",
				s.ModTime().UTC().Unix(), "",
			})
			return nil
		},
		Unsorted:            true,
		FollowSymbolicLinks: true,
	})
	util.Log("fsdb:", rt+":", "scan complete.")
}

func NewFiles(rows *sql.Rows) []itypes.File {
	r := []itypes.File{}
	for rows.Next() {
		r = append(r, itypes.ScanFile(rows))
	}
	rows.Close()
	return r
}

func insertFile(f *itypes.File) {
	oldF := NewFiles(db.FS.Build().Se("*").Fr(cTbl).Wh("path", f.Path).Exe())
	if len(oldF) > 0 {
		if oldF[0].ModTime == f.ModTime {
			// File exists and ModTime has not changed, skip
			return
		}
		// File exists but ModTime changed, updated
		s := strconv.FormatInt(oldF[0].ID, 10)
		db.FS.Build().Up(cTbl, "size", strconv.FormatInt(f.Size, 10)).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "mod_time", strconv.FormatInt(f.ModTime, 10)).Wh("id", s).Exe()
		return
	}
	// File does not exist, add
	db.FS.Build().Ins(cTbl).Lock()
	id := db.FS.QueryNextID(cTbl)
	db.FS.QueryPrepared(true, "insert into "+cTbl+" values (?,?,?,?,?)", id, f.Root, f.Path, f.Size, f.ModTime)
	db.FS.Build().Ins(cTbl).Unlock()
}
