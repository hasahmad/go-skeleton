package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
)

func (h Handlers) CreateAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		h.errors.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		h.errors.FailedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := h.models.Users.GetByEmail(r.Context(), input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			h.errors.InvalidCredentialsResponse(w, r)
			return
		default:
			h.errors.ServerErrorResponse(w, r, err)
			return
		}
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	if !match {
		h.errors.InvalidCredentialsResponse(w, r)
		return
	}

	token, err := h.models.Tokens.New(r.Context(), user.UserID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{"authentication_token": token}, nil)
	if err != nil {
		h.errors.ServerErrorResponse(w, r, err)
		return
	}
}
