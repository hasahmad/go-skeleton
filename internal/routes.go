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

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.controllers.HealthcheckHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.controllers.CreateAuthenticationTokenHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.controllers.RegisterUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.controllers.ActivateUserHandler)

	router.HandlerFunc(http.MethodGet, "/v1/users", app.middlewares.RequirePermission("users:list", app.controllers.ListUsersHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/:id", app.middlewares.RequirePermission("users:show", app.controllers.ShowUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/users/:id", app.middlewares.RequirePermission("users:edit", app.controllers.UpdateUserHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/:id", app.middlewares.RequirePermission("users:delete", app.controllers.DeleteUserHandler))

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.middlewares.Metrics(app.middlewares.RecoverPanic(app.middlewares.EnableCORS(app.middlewares.RateLimit(app.middlewares.Authenticate(router)))))
}
