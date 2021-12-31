package middlewares

import (
	"net/http"

	apicontext "github.com/hasahmad/go-skeleton/internal/api/context"
)

func (m *Middlewares) RequirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := apicontext.ContextGetUser(r)

		permissions, err := m.models.Permissions.GetAllForUser(r.Context(), user.UserID)
		if err != nil {
			m.errors.ServerErrorResponse(w, r, err)
			return
		}

		if !permissions.Include(code) {
			m.errors.NotPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return m.RequireActivatedUser(fn)
}
