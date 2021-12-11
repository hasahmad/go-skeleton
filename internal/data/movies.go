package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/hasahmad/greenlight/internal/validator"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Movie struct {
	TimeStampsModel
	SoftDeletableTimeStampModel
	ID        int64          `json:"id" db:"id"`
	CreatedAt pq.NullTime    `json:"created_at" db:"created_at"`
	Title     string         `json:"title" db:"title"`
	Year      int32          `json:"year,omitempty" db:"year"`
	Runtime   Runtime        `json:"runtime,omitempty" db:"runtime"`
	Genres    pq.StringArray `json:"genres,omitempty" db:"genres"`
	Version   int32          `json:"version" db:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive number")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

type MovieModel struct {
	DB        *sqlx.DB
	tableName string
}

func NewMovieModel(db *sqlx.DB) MovieModel {
	return MovieModel{
		DB:        db,
		tableName: "movies",
	}
}

func (m *MovieModel) Insert(ctx context.Context, movie *Movie) error {
	query, args, err := goqu.
		Insert(m.tableName).
		Rows(map[string]interface{}{
			"title":      movie.Title,
			"year":       movie.Year,
			"runtime":    movie.Runtime,
			"genres":     movie.Genres,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		}).
		Returning("id", "created_at", "version").
		ToSQL()
	if err != nil {
		return err
	}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m *MovieModel) Get(ctx context.Context, id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query, args, err := goqu.
		Select("*").
		From(m.tableName).
		Where(goqu.Ex{"id": id, "deleted_at": nil}).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var movie Movie
	err = m.DB.GetContext(ctx, &movie, query, args...)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func (m *MovieModel) Update(ctx context.Context, movie *Movie) error {
	query, args, err := goqu.
		Update(m.tableName).
		Set(map[string]interface{}{
			"title":      movie.Title,
			"year":       movie.Year,
			"runtime":    movie.Runtime,
			"genres":     movie.Genres,
			"version":    movie.Version + 1,
			"updated_at": time.Now(),
		}).
		Where(goqu.Ex{
			"id":         movie.ID,
			"version":    movie.Version,
			"deleted_at": nil,
		}).
		Returning("version").
		ToSQL()
	if err != nil {
		return err
	}

	err = m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m *MovieModel) Delete(ctx context.Context, id int64) error {
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
