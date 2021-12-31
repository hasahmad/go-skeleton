package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddPermissionsTable, downAddPermissionsTable)
}

func upAddPermissionsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS permissions (
		permission_id UUID PRIMARY KEY DEFAULT uuid_generate_v1(),
    	code varchar(150) NOT NULL,
		description text
	)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS users_permissions (
		user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
		permission_id UUID NOT NULL REFERENCES permissions ON DELETE CASCADE,
		PRIMARY KEY (user_id, permission_id)
	)
	`)
	return err
}

func downAddPermissionsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE users_permissions`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE permissions`)
	return err
}
