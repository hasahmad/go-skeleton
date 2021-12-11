package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/hasahmad/greenlight/internal/validator"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Movie struct {
	ID        int64          `json:"id" db:"id"`
	CreatedAt time.Time      `json:"-" db:"created_at"`
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

func (m *MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	return m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m *MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query, args, err := goqu.
		Select("*").
		From(m.tableName).
		Where(goqu.Ex{"id": id}).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var movie Movie
	err = m.DB.Get(&movie, query, args...)
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

func (m *MovieModel) Update(movie *Movie) error {
	query, args, err := goqu.
		Update(m.tableName).
		Set(map[string]interface{}{
			"title":   movie.Title,
			"year":    movie.Year,
			"runtime": movie.Runtime,
			"genres":  movie.Genres,
			"version": movie.Version + 1,
		}).
		Where(goqu.Ex{
			"id":      movie.ID,
			"version": movie.Version,
		}).
		Returning("version").
		ToSQL()
	if err != nil {
		return err
	}

	err = m.DB.QueryRow(query, args...).Scan(&movie.Version)
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

func (m *MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query, args, err := goqu.
		Delete(m.tableName).
		Where(goqu.Ex{
			"id": id,
		}).
		ToSQL()
	if err != nil {
		return err
	}

	result, err := m.DB.Exec(query, args...)
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
