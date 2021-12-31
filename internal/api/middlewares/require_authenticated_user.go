package middlewares

import (
	"net/http"

	apicontext "github.com/hasahmad/go-skeleton/internal/api/context"
)

func (m *Middlewares) RequireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := apicontext.ContextGetUser(r)

		if user.IsAnonymousUser() {
			m.errors.AuthenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middlewares) RequireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := apicontext.ContextGetUser(r)

		if !user.IsActive {
			m.errors.InactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return m.RequireAuthenticatedUser(fn)
}
