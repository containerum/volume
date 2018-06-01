package postgres

import (
	"git.containerum.net/ch/volume-manager/pkg/database"
	"github.com/go-pg/pg/orm"
)

type VolumeFilter database.VolumeFilter

func (f *VolumeFilter) Filter(q *orm.Query) (*orm.Query, error) {

	return q, nil
}
