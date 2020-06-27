package db

import (
	"database/sql"
	"strconv"

	dbstorage "github.com/nektro/go.dbstorage"
)

type Share struct {
	ID   int64  `json:"id"`
	Hash string `json:"hash" sqlite:"text"`
	Path string `json:"path" sqlite:"text"`
}

// Scan implements dbstorage.Scannable
func (v Share) Scan(rows *sql.Rows) dbstorage.Scannable {
	rows.Scan(&v.ID, &v.Hash, &v.Path)
	return &v
}

func (Share) ScanAll(q dbstorage.QueryBuilder) []*Share {
	arr := dbstorage.ScanAll(q, Share{})
	res := []*Share{}
	for _, item := range arr {
		o, ok := item.(*Share)
		if !ok {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (v *Share) i() string {
	return strconv.FormatInt(v.ID, 10)
}

func (Share) b() dbstorage.QueryBuilder {
	return DB.Build().Se("*").Fr(ctShare)
}

func (Share) All() []*Share {
	return Share{}.ScanAll(Share{}.b())
}

//
// searchers
//

func (Share) ByCode(c string) (*Share, bool) {
	ur, ok := dbstorage.ScanFirst(Share{}.b().Wh("hash", c), Share{}).(*Share)
	return ur, ok
}

//
// modifiers
//
