package fsdb

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"

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
			insertFile(&db.File{
				0, "0",
				rt,
				upathS, osPathname,
				s.Size(), "",
				s.ModTime().UTC().Unix(), "",
				"", "", "", "", "", "",
			})
			return nil
		},
		Unsorted:            true,
		FollowSymbolicLinks: true,
	})
	util.Log("fsdb:", rt+":", "scan complete.")
}

func insertFile(f *db.File) {
	oldF, ok := db.File{}.ByPath(f.Path)
	if ok {
		if oldF.ModTime == f.ModTime {
			// File exists and ModTime has not changed, skip
			return
		}
		// File exists but ModTime changed, updated
		s := oldF.IDS
		f.PopulateHashes()
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
	f.PopulateHashes()
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
