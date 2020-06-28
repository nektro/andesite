package fsdb

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"

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
			insertFile(&db.File{
				0,
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
		f.PopulateHashes(true)
		f.SetSize(f.Size)
		f.SetModTime(f.ModTime)
		return
	}
	// File does not exist, add
	if idata.Config.VerboseFS {
		util.Log("fsdb:", "add:", f.Path)
	}
	f.PopulateHashes(false)
	db.CreateFile(f.Root, f.Path, f.Size, f.ModTime, f.MD5, f.SHA1, f.SHA256, f.SHA512, f.SHA3, f.BLAKE2b)
}

func DeInit(mp map[string]string, rt string) {
	_, ok := mp[rt]
	if !ok {
		return
	}
	db.DropFilesFromRoot(rt)
	util.Log("fsdb:", rt+":", "removed.")
}
