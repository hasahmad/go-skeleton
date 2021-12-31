package apicontext

import (
	"context"
	"net/http"

	"github.com/hasahmad/go-skeleton/internal/data"
)

type ContextKey string

// Convert the string "user" to a contextKey type and assign it to the userContextKey
// constant. We'll use this constant as the key for getting and setting user information
// in the request context.
const userContextKey = ContextKey("user")

// The contextSetUser() method returns a new copy of the request with the provided
// User struct added to the context. Note that we use our userContextKey constant as the
// key.
func ContextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
