package middlewares

import (
	"errors"
	"net/http"
	"strings"

	apicontext "github.com/hasahmad/go-skeleton/internal/api/context"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
)

func (m *Middlewares) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		// if no authorization, set to anonymous user
		if authorizationHeader == "" {
			r = apicontext.ContextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// should be like: "Bearer JHFU876YGVGRUYJG..."
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			m.errors.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			m.errors.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := m.models.Users.GetForToken(r.Context(), data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				m.errors.InvalidAuthenticationTokenResponse(w, r)
				return
			default:
				m.errors.BadRequestResponse(w, r, err)
				return
			}
		}

		// set user and serve
		r = apicontext.ContextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}
