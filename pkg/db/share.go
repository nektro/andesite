package db

import (
	"database/sql"
	"strconv"

	"github.com/nektro/go-util/util"
	dbstorage "github.com/nektro/go.dbstorage"

	. "github.com/nektro/go-util/alias"
)

type Share struct {
	ID   int64  `json:"id"`
	Hash string `json:"hash" dbsorm:"1"`
	Path string `json:"path" dbsorm:"1"`
}

func CreateShare(pt string) *Share {
	dbstorage.InsertsLock.Lock()
	defer dbstorage.InsertsLock.Unlock()
	//
	id := db.QueryNextID(ctShare)
	hv := util.Hash("MD5", []byte(F("astheno.andesite.share.%s.%s", strconv.FormatInt(id, 10), T())))[:12]
	rv := &Share{id, hv, pt}
	db.Build().InsI(ctShare, rv).Exe()
	return rv
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
	return db.Build().Se("*").Fr(ctShare)
}

func (Share) All() []*Share {
	return Share{}.ScanAll(Share{}.b())
}

//
// searchers
//

func (Share) ByID(id int64) (*Share, bool) {
	ur, ok := dbstorage.ScanFirst(Share{}.b().Wh("id", strconv.FormatInt(id, 10)), Share{}).(*Share)
	return ur, ok
}

func (Share) ByCode(c string) (*Share, bool) {
	ur, ok := dbstorage.ScanFirst(Share{}.b().Wh("hash", c), Share{}).(*Share)
	return ur, ok
}

//
// modifiers
//

func (v *Share) SetPath(s string) {
	v.Path = s
	Up(v, db, ctShare, "path", s)
}

func (v *Share) Delete() {
	Del(v, db, ctShare)
}
