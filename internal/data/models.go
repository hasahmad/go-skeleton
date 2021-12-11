package data

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type TimeStampsModel struct {
	CreatedAt pq.NullTime `json:"created_at" db:"created_at"`
	UpdatedAt pq.NullTime `json:"updated_at" db:"updated_at"`
}

type SoftDeletableTimeStampModel struct {
	RemovedAt pq.NullTime `json:"created_at" db:"removed_at"`
}

type Models struct {
	Movies MovieModel
}

func NewModels(db *sqlx.DB) Models {
	return Models{
		Movies: NewMovieModel(db),
	}
}
