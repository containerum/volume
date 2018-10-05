package postgres

import (
	"context"

	"git.containerum.net/ch/volume-manager/pkg/database"
	"git.containerum.net/ch/volume-manager/pkg/errors"
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/pg"
	"github.com/sirupsen/logrus"
)

func (pgdb *PgDB) VolumeByLabel(ctx context.Context, nsID, label string) (ret model.Volume, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"ns_id": nsID,
		"label": label,
	}).Debugf("get volume by id")

	ret.NamespaceID = nsID
	ret.Label = label
	err = pgdb.db.Model(&ret).
		ColumnExpr("?TableAlias.*").
		Where("ns_id = ?ns_id").
		Where("label = ?label").
		Where("NOT deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("volume with name '%s' not exists", label)
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) UserVolumes(ctx context.Context, userID string) (ret []model.Volume, err error) {
	pgdb.log.WithField("user_id", userID).Debugf("get all user volumes")

	ret = make([]model.Volume, 0)

	err = pgdb.db.Model(&ret).
		Where("owner_user_id = ?", userID).
		Where("NOT deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) NamespaceVolumes(ctx context.Context, nsID string) (ret []model.Volume, err error) {
	pgdb.log.WithField("namespace_id", nsID).Debugf("get namespace volumes")

	ret = make([]model.Volume, 0)

	err = pgdb.db.Model(&ret).
		Where("ns_id = ?", nsID).
		Where("NOT deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) AllVolumes(ctx context.Context, filter database.VolumeFilter) (ret []model.Volume, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"filters": filter,
	}).Debugf("get all volumes")

	ret = make([]model.Volume, 0)

	f := VolumeFilter(filter)
	err = pgdb.db.Model(&ret).
		Apply(f.Filter).
		Select()
	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) CreateVolume(ctx context.Context, volume *model.Volume) error {
	pgdb.log.Debugf("create volume %+v", volume)

	_, err := pgdb.db.Model(volume).
		Returning("*").
		Insert()
	return pgdb.handleError(err)
}

func (pgdb *PgDB) DeleteVolume(ctx context.Context, volume *model.Volume) error {
	pgdb.log.Debugf("delete volume %+v", volume)

	result, err := pgdb.db.Model(volume).
		WherePK().
		Set("deleted = ?deleted").
		Set("delete_time = now()").
		Returning("*").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("volume %s not exists", volume.Label)
	}

	return nil
}

func (pgdb *PgDB) DeleteVolumes(ctx context.Context, volumes []model.Volume) error {
	pgdb.log.Debugf("delete volumes %+v", volumes)

	if len(volumes) == 0 {
		return nil
	}

	volIDs := make([]string, len(volumes))
	for i := range volumes {
		volIDs[i] = volumes[i].ID
	}

	volumes = nil
	_, err := pgdb.db.Model(&volumes).
		Where("id IN (?)", pg.In(volIDs)).
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}

	return nil
}

func (pgdb *PgDB) UpdateVolume(ctx context.Context, volume *model.Volume) error {
	pgdb.log.Debugf("update volume %+v", volume)

	result, err := pgdb.db.Model(volume).
		WherePK().
		Set("tariff_id = ?tariff_id").
		Set("capacity = ?capacity").
		Set("ns_id = ?ns_id").
		Set("access_mode = ?access_mode").
		Returning("*").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}

	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("volume %s not exists", volume.Label)
	}

	return nil
}
