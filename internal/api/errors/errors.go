package apierrors

import (
	"fmt"
	"net/http"

	"github.com/hasahmad/go-skeleton/internal/api/helpers"
	"github.com/sirupsen/logrus"
)

type ErrorResponses struct {
	logger *logrus.Logger
}

func New(logger *logrus.Logger) ErrorResponses {
	return ErrorResponses{
		logger: logger,
	}
}

func (e ErrorResponses) LogError(r *http.Request, err error) {
	e.logger.WithFields(logrus.Fields{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	}).Error(err)
}

func (e ErrorResponses) ErrorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	data := helpers.Envelope{"error": message}

	err := helpers.WriteJSON(w, status, data, nil)
	if err != nil {
		e.LogError(r, err)
		w.WriteHeader(500)
	}
}

func (e ErrorResponses) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.LogError(r, err)

	message := "the server encountered a problem and could not process your request"
	e.ErrorResponse(w, r, http.StatusInternalServerError, message)
}

func (e ErrorResponses) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	e.ErrorResponse(w, r, http.StatusNotFound, message)
}

func (e ErrorResponses) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	e.ErrorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (e ErrorResponses) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.ErrorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (e ErrorResponses) FailedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	e.ErrorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (e ErrorResponses) EditConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	e.ErrorResponse(w, r, http.StatusConflict, message)
}

func (e ErrorResponses) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	e.ErrorResponse(w, r, http.StatusTooManyRequests, message)
}

func (e ErrorResponses) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	e.ErrorResponse(w, r, http.StatusUnauthorized, message)
}

func (e ErrorResponses) InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	// help inform or remind the client that we expect them to authenticate using a bearer token.
	w.Header().Set("WWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	e.ErrorResponse(w, r, http.StatusUnauthorized, message)
}

func (e ErrorResponses) AuthenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	e.ErrorResponse(w, r, http.StatusUnauthorized, message)
}

func (e ErrorResponses) InactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	e.ErrorResponse(w, r, http.StatusForbidden, message)
}

func (e ErrorResponses) NotPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	e.ErrorResponse(w, r, http.StatusForbidden, message)
}
