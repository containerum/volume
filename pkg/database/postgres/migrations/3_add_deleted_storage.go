package migrations

import (
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := db.Model(&model.Storage{}).Exec( /* language=sql*/
			`ALTER TABLE "?TableName" 
				  		ADD COLUMN IF NOT EXISTS "deleted" Boolean NOT NULL,
				  		ADD COLUMN IF NOT EXISTS "delete_time" Timestamp With Time Zone;
`); err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		if _, err := db.Model(&model.Storage{}).Exec( /* language=sql*/
			`ALTER TABLE "?TableName" 
				  		DROP COLUMN IF EXISTS "deleted",
				  		DROP COLUMN IF EXISTS "delete_time";
`); err != nil {
			return err
		}
		return nil
	})
}
