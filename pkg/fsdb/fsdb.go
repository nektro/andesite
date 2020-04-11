package fsdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"
	"github.com/nektro/andesite/pkg/itypes"

	"github.com/karrick/godirwalk"
	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"
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
				hash("MD5", osPathname),
				hash("SHA1", osPathname),
				hash("SHA256", osPathname),
				hash("SHA512", osPathname),
				hash("SHA3_512", osPathname),
				hash("BLAKE2b_512", osPathname),
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
		db.FS.Build().Up(cTbl, "hash_md5", f.MD5).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "hash_sha1", f.SHA1).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "hash_sha256", f.SHA256).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "hash_sha512", f.SHA512).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "hash_sha3", f.SHA3).Wh("id", s).Exe()
		db.FS.Build().Up(cTbl, "hash_blake2b", f.BLAKE2b).Wh("id", s).Exe()
		return
	}
	// File does not exist, add
	dbstorage.InsertsLock.Lock()
	id := db.FS.QueryNextID(cTbl)
	if idata.Config.VerboseFS {
		util.Log("fsdb:", "add:", id, f.Path)
	}
	db.FS.Build().Ins(cTbl, id, f.Root, f.Path, f.Size, f.ModTime, f.MD5, f.SHA1, f.SHA256, f.SHA512, f.SHA3, f.BLAKE2b).Exe()
	dbstorage.InsertsLock.Unlock()
}

func DeInit(mp map[string]string, rt string) {
	_, ok := mp[rt]
	if !ok {
		return
	}
	db.FS.Build().Del("files").Wh("root", rt).Exe()
	util.Log("fsdb:", rt+":", "removed.")
}
