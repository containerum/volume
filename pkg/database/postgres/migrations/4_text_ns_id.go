package migrations

import (
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ALTER COLUMN ns_id TYPE TEXT USING (ns_id::TEXT);`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ALTER COLUMN ns_id TYPE UUID USING (ns_id::UUID);`)
		return err
	})
}
