package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAdduuidExtension, downAdduuidExtension)
}

func upAdduuidExtension(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	return err
}

func downAdduuidExtension(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
