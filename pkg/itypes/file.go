package itypes

import (
	"database/sql"
	"time"

	"github.com/nektro/go-util/util"
)

type File struct {
	ID       int64  `json:"id"`
	Root     string `json:"root" sqlite:"text"`
	Path     string `json:"path" sqlite:"text"`
	Size     int64  `json:"size" sqlite:"int"`
	SizeS    string `json:"html_size"`
	ModTime  int64  `json:"mod_time" sqlite:"int"`
	ModTimeS string `json:"html_modtime"`
}

func ScanFile(rows *sql.Rows) File {
	var v File
	rows.Scan(&v.ID, &v.Root, &v.Path, &v.Size, &v.ModTime)
	v.SizeS = util.ByteCountIEC(v.Size)
	v.ModTimeS = time.Unix(v.ModTime, -1).UTC().String()[:19]
	return v
}
