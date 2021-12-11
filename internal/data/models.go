package data

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type TimeStampsModel struct {
	CreatedAt NullTime `json:"created_at" db:"created_at"`
	UpdatedAt NullTime `json:"updated_at" db:"updated_at"`
}

type SoftDeletableTimeStampModel struct {
	RemovedAt NullTime `json:"-" db:"deleted_at"`
}

type Models struct {
	Movies MovieModel
}

func NewModels(db *sqlx.DB) Models {
	return Models{
		Movies: NewMovieModel(db),
	}
}
