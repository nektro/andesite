package fsdb

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nektro/andesite/pkg/db"
	"github.com/nektro/andesite/pkg/idata"

	"github.com/karrick/godirwalk"
	"github.com/nektro/go-util/util"
)

func Init(mp map[string]string, rt string) {
	bd, ok := mp[rt]
	if !ok {
		return
	}
	util.Log("fsdb:", rt+":", "scan begin...")
	start := time.Now()
	godirwalk.Walk(bd, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			fpathS, _ := filepath.Abs(osPathname)
			if strings.Contains(fpathS, "/.") {
				return nil
			}
			upathS := "/" + rt + strings.TrimPrefix(fpathS, bd)
			s, _ := os.Lstat(osPathname)
			if s.IsDir() {
				return nil
			}
			if s.Mode()&os.ModeSymlink != 0 {
				realpath, _ := filepath.EvalSymlinks(osPathname)
				if realpath == "" {
					util.LogError("fsdb:", rt+":", "symlink", osPathname, "is pointing to a non-existing file")
					return godirwalk.SkipThis
				}
				s, err := os.Lstat(realpath)
				if err != nil {
					util.LogError("fsdb:", rt+":", err)
					return nil
				}
				if s.IsDir() {
					return nil
				}
			}
			idata.HashingSem.Add()
			defer func() {
				defer idata.HashingSem.Done()
				insertFile(&db.File{
					0,
					rt,
					upathS, osPathname,
					s.Size(), "",
					s.ModTime().UTC().Unix(), "",
					"", "", "", "", "", "",
				})
			}()
			return nil
		},
		Unsorted:            true,
		FollowSymbolicLinks: true,
	})
	dur := time.Since(start)
	util.Log("fsdb:", rt+":", "scan completed in "+dur.String())
}

func insertFile(f *db.File) {
	oldF, ok := db.File{}.ByPath(f.Path)
	if ok {
		if oldF.ModTime == f.ModTime {
			// File exists and ModTime has not changed, skip
			if idata.Config.VerboseFS {
				util.Log("fsdb:", "skp:", f.Path)
			}
			return
		}
		// File exists but ModTime changed, updated
		f.PopulateHashes(true)
		f.SetSize(f.Size)
		f.SetModTime(f.ModTime)
		if idata.Config.VerboseFS {
			util.Log("fsdb:", "upd:", f.Path)
		}
		return
	}
	// File does not exist, add
	if idata.Config.Verbose {
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
