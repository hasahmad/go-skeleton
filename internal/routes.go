package internal

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) Routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.errors.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.errors.MethodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.handlers.HealthcheckHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.handlers.CreateAuthenticationTokenHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.handlers.RegisterUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.handlers.ActivateUserHandler)

	router.HandlerFunc(http.MethodGet, "/v1/users", app.middlewares.RequirePermission("users:list", app.handlers.ListUsersHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/:id", app.middlewares.RequirePermission("users:show", app.handlers.ShowUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/users/:id", app.middlewares.RequirePermission("users:edit", app.handlers.UpdateUserHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/:id", app.middlewares.RequirePermission("users:delete", app.handlers.DeleteUserHandler))

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.middlewares.Metrics(app.middlewares.RecoverPanic(app.middlewares.EnableCORS(app.middlewares.RateLimit(app.middlewares.Authenticate(router)))))
}
