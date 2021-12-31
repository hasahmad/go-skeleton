package controllers

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

func (ctrl Controllers) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
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
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = ctrl.models.Users.Insert(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = ctrl.models.Permissions.AddForUser(r.Context(), user.UserID, "movies:read")
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	token, err := ctrl.models.Tokens.New(r.Context(), user.UserID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	// Send the welcome email in the background
	helpers.Background(ctrl.logger, ctrl.wg, func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.UserID,
		}
		err = ctrl.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			ctrl.logger.Error(err)
		}
	})

	err = helpers.WriteJSON(w, http.StatusAccepted, helpers.Envelope{"user": user}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := ctrl.models.Users.GetForToken(r.Context(), data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			ctrl.errors.FailedValidationResponse(w, r, v.Errors)
			return
		default:
			ctrl.errors.BadRequestResponse(w, r, err)
			return
		}
	}

	user.IsActive = true
	err = ctrl.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			ctrl.errors.EditConflictResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	// If everything went successfully, then we delete all activation tokens for the
	// user.
	err = ctrl.models.Tokens.DeleteAllForUser(r.Context(), data.ScopeActivation, user.UserID)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	user, err := ctrl.models.Users.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			ctrl.errors.NotFoundResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
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
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
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

	users, metadata, err := ctrl.models.Users.GetAll(r.Context(), where, input.Filters)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"metadata": metadata, "users": users}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	user, err := ctrl.models.Users.Get(r.Context(), id)
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
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Username  string `json:"username"`
	}

	err = helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
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
		ctrl.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	err = ctrl.models.Users.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			ctrl.errors.EditConflictResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}

func (ctrl Controllers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.ReadUUIDParam(r)
	if err != nil {
		ctrl.errors.NotFoundResponse(w, r)
		return
	}

	err = ctrl.models.Users.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			ctrl.errors.NotFoundResponse(w, r)
		default:
			ctrl.errors.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"message": "user successfully deleted"}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}
