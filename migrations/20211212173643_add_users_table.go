package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddUsersTable, downAddUsersTable)
}

func upAddUsersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		"user_id" UUID NOT NULL DEFAULT uuid_generate_v1(),
		"username" varchar(150) NOT NULL,
		"password" varchar(128) NOT NULL,
		"first_name" varchar(30),
		"last_name" varchar(150),
		"email" varchar(254),
		"is_superuser" bool NOT NULL DEFAULT False,
		"is_staff" bool NOT NULL DEFAULT False,
		"is_active" bool NOT NULL DEFAULT False,
		"last_login" timestamptz,
		"created_at" timestamptz DEFAULT NOW(),
		"updated_at" timestamptz DEFAULT NOW(),
		"deleted_at" timestamptz,
		"version" integer NOT NULL DEFAULT 1,
		CONSTRAINT "users_pkey" PRIMARY KEY ("user_id")
	)`)
	return err
}

func downAddUsersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE users`)
	return err
}
