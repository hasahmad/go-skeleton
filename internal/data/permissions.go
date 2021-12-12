package data

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Permission struct {
	ID   int64  `json:"id" db:"id"`
	Code string `json:"code" db:"code"`
}

type Permissions []string

func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}

	return false
}

type PermissionModel struct {
	DB        *sqlx.DB
	tableName string
}

func NewPermissionModel(db *sqlx.DB) PermissionModel {
	return PermissionModel{
		DB:        db,
		tableName: "permissions",
	}
}

func (m PermissionModel) GetAllForUser(ctx context.Context, userID int64) (Permissions, error) {
	query, args, err := goqu.
		Select(goqu.I("p.code")).
		From(goqu.T(m.tableName).As("p")).
		Join(
			goqu.T("users_permissions").As("up"),
			goqu.On(goqu.I("up.permission_id").Eq(goqu.I("p.id"))),
		).
		Where(goqu.Ex{"up.user_id": userID}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (m PermissionModel) AddForUser(ctx context.Context, userID int64, codes ...string) error {
	query, args, err := goqu.
		Insert("users_permissions").
		FromQuery(goqu.
			Select(goqu.V(userID).As("user_id"), goqu.I("permissions.id").As("permission_id")).
			From(m.tableName).
			Where(goqu.L("permissions.code = ?", goqu.Any(pq.Array(codes))))).
		ToSQL()

	if err != nil {
		return err
	}

	_, err = m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
