package migrations

import (
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP CONSTRAINT IF EXISTS unique_label_namespace;
					ALTER TABLE "?TableName" ADD CONSTRAINT unique_label_namespace UNIQUE (ns_id, label)`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP CONSTRAINT IF EXISTS unique_label_namespace`)
		return err
	})
}
