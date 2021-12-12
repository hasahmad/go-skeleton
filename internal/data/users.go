package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/hasahmad/greenlight/internal/validator"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	TimeStampsModel
	SoftDeletableTimeStampModel
	ID        int64    `json:"id" db:"id"`
	Name      string   `json:"name" db:"name"`
	Email     string   `json:"email" db:"email"`
	Password  password `json:"-" db:"password_hash"`
	Activated bool     `json:"activated" db:"activated"`
	Version   int      `json:"-" db:"version"`
}

type password struct {
	plaintext *string
	hash      []byte
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
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

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
			"created_at":    time.Now(),
			"updated_at":    time.Now(),
			"name":          user.Name,
			"email":         user.Email,
			"password_hash": user.Password.hash,
			"activated":     user.Activated,
		}).
		Returning("id", "created_at", "version").
		ToSQL()

	if err != nil {
		return err
	}

	err = m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
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

func (m UserModel) Update(ctx context.Context, user *User) error {
	data := map[string]interface{}{
		"activated":  user.Activated,
		"version":    user.Version + 1,
		"updated_at": time.Now(),
	}
	if user.Password.hash != nil {
		data["password_hash"] = user.Password.hash
	}
	if user.Email != "" {
		data["email"] = user.Email
	}
	if user.Name != "" {
		data["name"] = user.Name
	}

	query, args, err := goqu.
		Update(m.tableName).
		Set(data).
		Where(goqu.Ex{
			"id":         user.ID,
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

func (m UserModel) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query, args, err := goqu.
		Update(m.tableName).
		Set(
			goqu.Record{"deleted_at": time.Now()},
		).
		Where(goqu.Ex{
			"id": id,
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
