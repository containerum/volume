package migrations

import (
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP CONSTRAINT IF EXISTS volumes_label_owner_user_id_key`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP CONSTRAINT IF EXISTS volumes_label_owner_user_id_key;
					ALTER TABLE "?TableName" ADD CONSTRAINT volumes_label_owner_user_id_key UNIQUE (owner_user_id, label)`)
		return err
	})
}
