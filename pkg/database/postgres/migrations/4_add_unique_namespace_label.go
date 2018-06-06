package migrations

import (
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`CREATE UNIQUE INDEX IF NOT EXISTS unique_label_namespace  ON "?TableName" (ns_id, label) WHERE NOT deleted`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`DROP INDEX IF EXISTS unique_label_namespace`)
		return err
	})
}
