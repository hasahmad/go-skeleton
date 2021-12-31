package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
)

func (ctrl Controllers) CreateMovieHandler(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = ctrl.models.Movies.Insert(r.Context(), movie)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location: ", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"movie": movie}, headers)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) ShowMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	movie, err := ctrl.models.Movies.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			ctrl.errors.NotFoundResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"movie": movie}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) UpdateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	movie, err := ctrl.models.Movies.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			ctrl.errors.NotFoundResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}

	err = helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Valid() {
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = ctrl.models.Movies.Update(r.Context(), movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			ctrl.errors.EditConflictResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"movie": movie}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) DeleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	err = ctrl.models.Movies.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			ctrl.errors.NotFoundResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Title, _ = helpers.ReadString(qs, "title", "")
	input.Genres, _ = helpers.ReadCSV(qs, "genres", []string{})
	input.Filters.Page, _ = helpers.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize, _ = helpers.ReadInt(qs, "page_size", 20, v)
	input.Filters.Sort, _ = helpers.ReadString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	movies, metadata, err := ctrl.models.Movies.GetAll(r.Context(), input.Title, input.Genres, input.Filters)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"metadata": metadata, "movies": movies}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}
