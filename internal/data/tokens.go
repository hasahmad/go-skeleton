package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/hasahmad/greenlight/internal/validator"
	"github.com/jmoiron/sqlx"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	Plaintext string
	Hash      []byte
	UserID    int64
	Expiry    time.Time
	Scope     string
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Encode the byte slice to a base-32-encoded string and assign it to the token
	// Plaintext field.
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB        *sqlx.DB
	tableName string
}

func NewTokenModel(db *sqlx.DB) TokenModel {
	return TokenModel{
		DB:        db,
		tableName: "tokens",
	}
}

func (m TokenModel) New(ctx context.Context, userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(ctx, token)
	return token, err
}

func (m TokenModel) Insert(ctx context.Context, token *Token) error {
	query, args, err := goqu.
		Insert(m.tableName).
		Rows(map[string]interface{}{
			"hash":    token.Hash,
			"user_id": token.UserID,
			"expiry":  token.Expiry,
			"scope":   token.Scope,
		}).
		ToSQL()
	if err != nil {
		return err
	}

	_, err = m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteAllForUser(ctx context.Context, scope string, userID int64) error {
	query, args, err := goqu.
		Delete(m.tableName).
		Where(goqu.Ex{
			"user_id": userID,
			"scope":   scope,
		}).
		ToSQL()
	if err != nil {
		return err
	}

	_, err = m.DB.ExecContext(ctx, query, args...)
	return err
}
