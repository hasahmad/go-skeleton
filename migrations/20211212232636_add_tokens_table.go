package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddTokensTable, downAddTokensTable)
}

func upAddTokensTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS tokens (
		hash bytea PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
		expiry timestamp(0) with time zone NOT NULL,
		scope text NOT NULL
	)
	`)
	return err
}

func downAddTokensTable(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE tokens`)
	return err
}
