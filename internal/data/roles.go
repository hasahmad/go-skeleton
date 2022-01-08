package data

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Role struct {
	RoleID uuid.UUID `json:"role_id" db:"role_id"`
	Code   string    `json:"code" db:"code"`
}

type Roles []string

func (r Roles) Include(code string) bool {
	for i := range r {
		if code == r[i] {
			return true
		}
	}

	return false
}

func (r Roles) IncludeMultiple(codes []string, any bool) bool {
	var count int = 0
	for i := range r {
		for j := range codes {
			if codes[j] == r[i] {
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

type RoleModel struct {
	DB        *sqlx.DB
	tableName string
}

func NewRoleModel(db *sqlx.DB) RoleModel {
	return RoleModel{
		DB:        db,
		tableName: "roles",
	}
}

func (m RoleModel) GetAllForUser(ctx context.Context, userID uuid.UUID) (Roles, error) {
	query, args, err := goqu.
		Select(goqu.I("p.code")).
		From(goqu.T(m.tableName).As("p")).
		Join(
			goqu.T("users_roles").As("up"),
			goqu.On(goqu.I("up.role_id").Eq(goqu.I("p.permission_id"))),
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

	var roles Roles

	for rows.Next() {
		var role string
		err := rows.Scan(&role)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

func (m RoleModel) AddForUser(ctx context.Context, userID uuid.UUID, codes ...string) error {
	query, args, err := goqu.
		Insert("users_roles").
		FromQuery(goqu.
			Select(goqu.V(userID).As("user_id"), goqu.I("roles.role_id").As("role_id")).
			From(m.tableName).
			Where(goqu.L("roles.code = ?", goqu.Any(pq.Array(codes))))).
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
