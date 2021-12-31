package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/hasahmad/go-skeleton/internal/validator"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/guregu/null.v4"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

var AnonymousUser = &User{}

type User struct {
	TimeStampsModel
	SoftDeletableTimeStampModel
	UserID      uuid.UUID   `json:"user_id" db:"user_id"`
	FirstName   string      `json:"first_name" db:"first_name"`
	LastName    null.String `json:"last_name" db:"last_name"`
	Username    null.String `json:"username" db:"username"`
	Email       string      `json:"email" db:"email"`
	Password    password    `json:"-" db:"password"`
	IsActive    bool        `json:"is_active" db:"is_active"`
	IsStaff     bool        `json:"is_staff" db:"is_staff"`
	IsSuperuser bool        `json:"is_superuser" db:"is_superuser"`
	LastLogin   null.Time   `json:"last_login" db:"last_login"`
	Version     int         `json:"-" db:"version"`
}

func (u *User) IsAnonymousUser() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Scan(value interface{}) error {
	if value == nil {
		p.plaintext, p.hash = nil, nil
		return nil
	}
	p.plaintext = nil
	v, ok := value.([]byte)
	if !ok {
		// most likely a string
		vstr, ok := value.(string)
		if !ok {
			return fmt.Errorf("unable to convert password hash")
		} else {
			p.hash = []byte(vstr)
		}
	} else {
		p.hash = v
	}

	return nil
}

func (p password) Value() (driver.Value, error) {
	if p.hash == nil {
		return nil, nil
	}
	return p.hash, nil
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.FirstName != "", "first_name", "must be provided")
	v.Check(len(user.FirstName) <= 500, "first_name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserModel struct {
	DB        *sqlx.DB
	tableName string
}

func NewUserModel(db *sqlx.DB) UserModel {
	return UserModel{
		DB:        db,
		tableName: "users",
	}
}

func (m UserModel) Insert(ctx context.Context, user *User) error {
	query, args, err := goqu.
		Insert(m.tableName).
		Rows(map[string]interface{}{
			"created_at":   time.Now(),
			"updated_at":   time.Now(),
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"email":        user.Email,
			"password":     user.Password.hash,
			"is_active":    user.IsActive,
			"is_staff":     user.IsStaff,
			"is_superuser": user.IsSuperuser,
			"username":     user.Username,
			"last_login":   user.LastLogin,
		}).
		Returning("user_id", "created_at", "updated_at", "version").
		ToSQL()

	if err != nil {
		return err
	}

	err = m.DB.QueryRowContext(ctx, query, args...).Scan(&user.UserID, &user.CreatedAt, &user.UpdatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	query, args, err := goqu.
		Select("*").
		From(m.tableName).
		Where(goqu.Ex{"user_id": id, "deleted_at": nil}).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var user User
	err = m.DB.GetContext(ctx, &user, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) GetByEmail(ctx context.Context, email string) (*User, error) {
	query, args, err := goqu.
		Select("*").
		From(m.tableName).
		Where(goqu.Ex{"email": email, "deleted_at": nil}).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var user User
	err = m.DB.GetContext(ctx, &user, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query, args, err := goqu.
		Select(goqu.I("u.*")).
		From(goqu.T(m.tableName).As("u")).
		Join(goqu.T("tokens").As("t"), goqu.On(
			goqu.I("t.user_id").Eq(goqu.I("u.user_id")))).
		Where(goqu.Ex{
			"t.hash":       tokenHash[:],
			"t.scope":      tokenScope,
			"u.deleted_at": nil,
		}).
		Where(goqu.I("t.expiry").Gt(time.Now())).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var user User
	err = m.DB.GetContext(ctx, &user, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) Update(ctx context.Context, user *User) error {
	data := map[string]interface{}{
		"is_active":  user.IsActive,
		"version":    user.Version + 1,
		"updated_at": time.Now(),
	}
	if user.Password.hash != nil {
		data["password"] = user.Password.hash
	}
	if user.Email != "" {
		data["email"] = user.Email
	}
	if user.FirstName != "" {
		data["first_name"] = user.FirstName
	}
	if user.LastName.Valid {
		data["last_name"] = user.LastName
	}
	if user.Username.Valid {
		data["username"] = user.Username
	}
	if user.IsSuperuser {
		data["is_superuser"] = user.IsSuperuser
	}

	query, args, err := goqu.
		Update(m.tableName).
		Set(data).
		Where(goqu.Ex{
			"user_id":    user.UserID,
			"version":    user.Version,
			"deleted_at": nil,
		}).
		Returning("version").
		ToSQL()
	if err != nil {
		return err
	}

	err = m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := goqu.
		Update(m.tableName).
		Set(
			goqu.Record{"deleted_at": time.Now()},
		).
		Where(goqu.Ex{
			"user_id": id,
		}).
		ToSQL()
	if err != nil {
		return err
	}

	result, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m UserModel) GetAll(ctx context.Context, wheres []goqu.Expression, filters Filters) ([]*User, Metadata, error) {
	sel := goqu.Select(
		goqu.COUNT("*").Over(goqu.W()),
		"user_id", "created_at", "updated_at",
		"email", "username",
		"first_name", "last_name",
		"is_active", "is_staff", "is_superuser", "version",
	).
		From(m.tableName).
		Where(goqu.Ex{"deleted_at": nil})

	for i := range wheres {
		sel.Where(wheres[i])
	}

	if filters.Sort != "" {
		if filters.sortDirection() == "DESC" {
			sel = sel.Order(goqu.I(filters.sortColumn()).Desc())
		} else {
			sel = sel.Order(goqu.I(filters.sortColumn()).Asc())
		}
	}

	if filters.limit() > 0 && filters.Page > 0 {
		sel = sel.Limit(uint(filters.limit())).
			Offset(uint(filters.offset()))
	}

	query, args, err := sel.ToSQL()
	if err != nil {
		return nil, Metadata{}, err
	}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrRecordNotFound
		default:
			return nil, Metadata{}, err
		}
	}

	defer rows.Close()

	// Declare a totalRecords variable.
	totalRecords := 0
	users := []*User{}

	for rows.Next() {
		var user User

		err := rows.Scan(
			&totalRecords, // Scan the count from the window function into totalRecords.
			&user.UserID,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Email,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.IsActive,
			&user.IsStaff,
			&user.IsSuperuser,
			&user.Version,
		)
		if err != nil {
			return nil, Metadata{}, err // Update this to return an empty Metadata struct.
		}

		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}

	return users, calculateMetadata(totalRecords, filters.Page, filters.PageSize), nil
}
