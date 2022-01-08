package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/hasahmad/go-skeleton/internal/api/helpers"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
	"gopkg.in/guregu/null.v4"
)

func (h Handlers) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		h.errors.BadRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		FirstName:   input.FirstName,
		LastName:    null.StringFrom(input.LastName),
		Email:       input.Email,
		Username:    null.StringFrom(input.Username),
		IsActive:    false,
		IsStaff:     false,
		IsSuperuser: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		h.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.models.Users.Insert(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			h.errors.FailedValidationResponse(w, r, v.Errors)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	// add initial user role once registered
	err = h.models.Roles.AddForUser(r.Context(), user.UserID, "user")
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	token, err := h.models.Tokens.New(r.Context(), user.UserID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	// Send the welcome email in the background
	helpers.Background(h.logger, h.wg, func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.UserID,
		}
		err = h.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			h.logger.Error(err)
		}
	})

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}

func (h Handlers) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		h.errors.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		h.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.models.Users.GetForToken(r.Context(), data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			h.errors.FailedValidationResponse(w, r, v.Errors)
			return
		default:
			h.errors.BadRequestResponse(w, r, err)
			return
		}
	}

	user.IsActive = true
	err = h.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			h.errors.EditConflictResponse(w, r)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	// If everything went successfully, then we delete all activation tokens for the
	// user.
	err = h.models.Tokens.DeleteAllForUser(r.Context(), data.ScopeActivation, user.UserID)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}

func (h Handlers) ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		h.errors.NotFoundResponse(w, r)
		return
	}

	user, err := h.models.Users.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			h.errors.NotFoundResponse(w, r)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}

func (h Handlers) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username  string
		Email     string
		FirstName string
		LastName  string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.FirstName, _ = helpers.ReadString(qs, "first_name", "")
	input.LastName, _ = helpers.ReadString(qs, "last_name", "")
	input.Username, _ = helpers.ReadString(qs, "username", "")
	input.Email, _ = helpers.ReadString(qs, "email", "")

	input.Filters.Page, _ = helpers.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize, _ = helpers.ReadInt(qs, "page_size", 20, v)
	input.Filters.Sort, _ = helpers.ReadString(qs, "sort", "user_id")
	input.Filters.SortSafelist = []string{
		"user_id", "username", "email", "first_name", "last_name",
		"-user_id", "-username", "-email", "-first_name", "-last_name",
	}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		h.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	where := []goqu.Expression{}
	if input.FirstName != "" {
		where = append(where, goqu.Ex{"first_name": input.FirstName})
	}
	if input.LastName != "" {
		where = append(where, goqu.Ex{"last_name": input.FirstName})
	}
	if input.Username != "" {
		where = append(where, goqu.Ex{"username": input.Username})
	}
	if input.Email != "" {
		where = append(where, goqu.Ex{"email": input.Email})
	}

	users, metadata, err := h.models.Users.GetAll(r.Context(), where, input.Filters)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"metadata": metadata, "users": users}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}

func (h Handlers) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		h.errors.NotFoundResponse(w, r)
		return
	}

	user, err := h.models.Users.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			h.errors.NotFoundResponse(w, r)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Username  string `json:"username"`
	}

	err = helpers.ReadJSON(w, r, &input)
	if err != nil {
		h.errors.BadRequestResponse(w, r, err)
		return
	}

	if input.FirstName != "" {
		user.FirstName = input.FirstName
	}
	if input.LastName != "" {
		user.LastName = null.StringFrom(input.LastName)
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Username != "" {
		user.Username = null.StringFrom(input.Username)
	}

	v := validator.New()

	if data.ValidateEmail(v, input.Email); !v.Valid() {
		h.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = h.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			h.errors.EditConflictResponse(w, r)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}

func (h Handlers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		h.errors.NotFoundResponse(w, r)
		return
	}

	err = h.models.Users.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			h.errors.NotFoundResponse(w, r)
		default:
			h.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"message": "user successfully deleted"}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
	}
}
