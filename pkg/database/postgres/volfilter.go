package postgres

import (
	"git.containerum.net/ch/volume-manager/pkg/database"
	"github.com/go-pg/pg/orm"
)

type VolumeFilter database.VolumeFilter

func (f *VolumeFilter) Filter(q *orm.Query) (*orm.Query, error) {
	if f.NotDeleted {
		q = q.Where("NOT ?TableAlias.deleted")
	}
	if f.Deleted {
		q = q.Where("?TableAlias.deleted")
	}

	if f.PerPage > 0 {
		pager := orm.Pager{Limit: f.PerPage}
		pager.SetPage(f.Page)
		q = q.Apply(pager.Paginate)
	}

	return q, nil
}
