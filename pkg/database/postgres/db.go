package postgres

import (
	"io"
	"time"

	"git.containerum.net/ch/volume-manager/pkg/database"
	"git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

type PgDB struct {
	db  orm.DB
	log *cherrylog.LogrusAdapter
}

type transactional interface {
	RunInTransaction(fn func(*pg.Tx) error) error
}

func (pgdb *PgDB) handleError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *cherry.Err:
		return err
	default:
		return errors.ErrInternal().Log(err, pgdb.log)
	}
}

func (pgdb *PgDB) Transactional(fn func(tx database.DB) error) error {
	entry := cherrylog.NewLogrusAdapter(pgdb.log.WithField("transaction_id", time.Now().UTC().Unix()))
	dtx := &PgDB{log: entry}
	err := pgdb.db.(transactional).RunInTransaction(func(tx *pg.Tx) error {
		dtx.db = tx
		return fn(dtx)
	})

	return dtx.handleError(err)
}

func (pgdb *PgDB) Close() error {
	if cl, ok := pgdb.db.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}
