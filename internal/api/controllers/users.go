package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
)

func (ctrl Controllers) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := helpers.ReadJSON(w, r, &input)
	if err != nil {
		ctrl.errors.BadRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
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

	err = ctrl.models.Permissions.AddForUser(r.Context(), user.ID, "movies:read")
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	token, err := ctrl.models.Tokens.New(r.Context(), user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	// Send the welcome email in the background
	helpers.Background(ctrl.logger, ctrl.wg, func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
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

	user.Activated = true
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
	err = ctrl.models.Tokens.DeleteAllForUser(r.Context(), data.ScopeActivation, user.ID)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
		return
	}

	err = helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{"user": user}, nil)
	if err != nil {
		ctrl.errors.ServerErrorResponse(w, r, err)
	}
}
