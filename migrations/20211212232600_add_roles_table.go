package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upAddRolesTable, downAddRolesTable)
}

func upAddRolesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS roles (
		role_id UUID PRIMARY KEY DEFAULT uuid_generate_v1(),
    	code varchar(150) NOT NULL,
		description text
	)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS users_roles (
		user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
		role_id UUID NOT NULL REFERENCES roles ON DELETE CASCADE,
		PRIMARY KEY (user_id, role_id)
	)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS roles_permissions (
		role_id UUID NOT NULL REFERENCES roles ON DELETE CASCADE,
		permission_id UUID NOT NULL REFERENCES permissions ON DELETE CASCADE,
		PRIMARY KEY (role_id, permission_id)
	)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT INTO roles (code) VALUES
	('user'),
	('subscriber'),
	('manager'),
	('admin')
	`)
	return err
}

func downAddRolesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE roles_permissions`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE users_roles`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DROP TABLE roles`)
	return err
}
