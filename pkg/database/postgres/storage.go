package postgres

import (
	"context"

	"git.containerum.net/ch/volume-manager/pkg/errors"
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/pg"
)

func (pgdb *PgDB) CreateStorage(ctx context.Context, storage *model.Storage) error {
	pgdb.log.Debugf("create storage %+v", storage)

	storage.Used = 0
	_, err := pgdb.db.Model(storage).
		Returning("*").
		Insert()
	return pgdb.handleError(err)
}

func (pgdb *PgDB) StorageByName(ctx context.Context, name string) (ret model.Storage, err error) {
	pgdb.log.WithField("name", name).Debugf("get storage by name")

	err = pgdb.db.Model(&ret).
		Where("name = ?", name).
		Where("NOT deleted").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("storage %s not exists", name)
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) AllStorages(ctx context.Context) (ret []model.Storage, err error) {
	pgdb.log.Debugf("get storage list")

	err = pgdb.db.Model(&ret).
		Where("NOT deleted").
		Select()
	err = pgdb.handleError(err)
	return
}

func (pgdb *PgDB) UpdateStorage(ctx context.Context, name string, storage model.Storage) error {
	pgdb.log.WithField("name", name).Debugf("update storage to %+v", storage)

	if storage.Name != name {
		cnt, err := pgdb.db.Model(&storage).
			WherePK().
			Count()
		if err != nil {
			return pgdb.handleError(err)
		}
		if cnt > 0 {
			return errors.ErrResourceAlreadyExists().AddDetailF("storage %s already exists", storage.Name)
		}
	}

	result, err := pgdb.db.Model(&storage).
		Where("name = ?", name).
		Set("name = ?name").
		Set("size = ?size").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}
	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("storage %s not exists", storage.Name)
	}
	return nil
}

func (pgdb *PgDB) DeleteStorage(ctx context.Context, storage *model.Storage) error {
	pgdb.log.WithField("name", storage.Name).Debugf("delete storage")

	result, err := pgdb.db.Model(storage).WherePK().
		Set("deleted = TRUE").
		Set("delete_time = now()").
		Returning("*").
		Update()
	if err != nil {
		return pgdb.handleError(err)
	}
	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("storage %s not exists", storage.Name)
	}
	return nil
}

func (pgdb *PgDB) LeastUsedStorage(ctx context.Context, minFree int) (ret model.Storage, err error) {
	pgdb.log.WithField("min_free", minFree).Debugf("get least used storage with constraint")

	err = pgdb.db.Model(&ret).
		Where("size - used >= ?", minFree).
		Where("NOT deleted").
		OrderExpr("used ASC").
		First()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrNoFreeStorages()
	default:
		err = pgdb.handleError(err)
	}

	return
}
