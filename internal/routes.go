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

	router.HandlerFunc(http.MethodGet, "/v1/movies", app.middlewares.RequirePermission("movies:read", app.controllers.ListMoviesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.middlewares.RequirePermission("movies:write", app.controllers.CreateMovieHandler))
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.middlewares.RequirePermission("movies:read", app.controllers.ShowMovieHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.middlewares.RequirePermission("movies:write", app.controllers.UpdateMovieHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.middlewares.RequirePermission("movies:write", app.controllers.DeleteMovieHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.controllers.RegisterUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.controllers.ActivateUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.controllers.CreateAuthenticationTokenHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.middlewares.Metrics(app.middlewares.RecoverPanic(app.middlewares.EnableCORS(app.middlewares.RateLimit(app.middlewares.Authenticate(router)))))
}
