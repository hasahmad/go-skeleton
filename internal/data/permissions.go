package data

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Permission struct {
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	Code         string    `json:"code" db:"code"`
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

func (p Permissions) IncludeMultiple(codes []string, any bool) bool {
	var count int = 0
	for i := range p {
		for j := range codes {
			if codes[j] == p[i] {
				count += 1
				if any {
					return true
				}
			}
		}
	}

	if any {
		return false
	}

	return count == len(codes)
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

func (m PermissionModel) GetAllForUser(ctx context.Context, userID uuid.UUID) (Permissions, error) {
	query, args, err := goqu.
		Select(goqu.I("p.code")).
		From(goqu.T(m.tableName).As("p")).
		LeftJoin(
			goqu.T("users_permissions").As("up"),
			goqu.On(goqu.I("up.permission_id").Eq(goqu.I("p.permission_id"))),
		).
		LeftJoin(
			goqu.T("roles_permissions").As("rp"),
			goqu.On(goqu.I("rp.permission_id").Eq(goqu.I("p.permission_id"))),
		).
		LeftJoin(
			goqu.T("users_roles").As("ur"),
			goqu.On(goqu.I("ur.role_id").Eq(goqu.I("rp.role_id"))),
		).
		Where(goqu.ExOr{"up.user_id": userID, "ur.user_id": userID}).
		GroupBy(goqu.I("p.permission_id")).
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

func (m PermissionModel) AddForUser(ctx context.Context, userID uuid.UUID, codes ...string) error {
	query, args, err := goqu.
		Insert("users_permissions").
		FromQuery(goqu.
			Select(goqu.V(userID).As("user_id"), goqu.I("permissions.permission_id").As("permission_id")).
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

func (m PermissionModel) AddForRole(ctx context.Context, roleID uuid.UUID, codes ...string) error {
	query, args, err := goqu.
		Insert("roles_permissions").
		FromQuery(goqu.
			Select(goqu.V(roleID).As("role_id"), goqu.I("permissions.permission_id").As("permission_id")).
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
