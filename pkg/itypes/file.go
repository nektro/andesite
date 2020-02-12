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
	MD5      string `json:"hash_md5" sqlite:"text"`
	SHA1     string `json:"hash_sha1" sqlite:"text"`
	SHA256   string `json:"hash_sha256" sqlite:"text"`
	SHA512   string `json:"hash_sha512" sqlite:"text"`
	SHA3     string `json:"hash_sha3" sqlite:"text"`
	BLAKE2b  string `json:"hash_blake2b" sqlite:"text"`
}

func ScanFile(rows *sql.Rows) File {
	var v File
	rows.Scan(&v.ID, &v.Root, &v.Path, &v.Size, &v.ModTime, &v.MD5, &v.SHA1, &v.SHA256, &v.SHA512, &v.SHA3, &v.BLAKE2b)
	v.SizeS = util.ByteCountIEC(v.Size)
	v.ModTimeS = time.Unix(v.ModTime, -1).UTC().String()[:19]
	return v
}
