package middlewares

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	apicontext "github.com/hasahmad/go-skeleton/internal/api/context"
	apierrors "github.com/hasahmad/go-skeleton/internal/api/errors"
	"github.com/hasahmad/go-skeleton/internal/config"
	"github.com/hasahmad/go-skeleton/internal/data"
	"github.com/hasahmad/go-skeleton/internal/validator"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

type Middlewares struct {
	logger *logrus.Logger
	cfg    config.Config
	errors apierrors.ErrorResponses
	models data.Models
}

func New(logger *logrus.Logger, cfg config.Config, errors apierrors.ErrorResponses, models data.Models) Middlewares {
	return Middlewares{
		logger: logger,
		cfg:    cfg,
		errors: errors,
		models: models,
	}
}

func (m *Middlewares) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function (which will always be run in the event of a panic
		// as Go unwinds the stack).
		defer func() {
			// Use the builtin recover function to check if there has been a panic or
			// not.
			if err := recover(); err != nil {
				// If there was a panic, set a "Connection: close" header on the
				// response. This acts as a trigger to make Go's HTTP server
				// automatically close the current connection after a response has been
				// sent.
				w.Header().Set("Connection", "close")

				m.errors.ServerErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (m *Middlewares) RateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mu.Lock()

			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			// Importantly, unlock the mutex when the cleanup is complete.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.cfg.Limiter.Enabled {
			ip := realip.FromRequest(r)

			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()

			// Check to see if the IP address already exists in the map. If it doesn't, then
			// initialize a new rate limiter and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(m.cfg.Limiter.RPS), m.cfg.Limiter.Burst),
				}
			}

			clients[ip].lastSeen = time.Now()

			// If the request isn't allowed, unlock the mutex and
			// send a 429 Too Many Requests response, just like before.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				m.errors.RateLimitExceededResponse(w, r)
				return
			}

			// Very importantly, unlock the mutex before calling the next handler in the
			// chain. Notice that NOT using defer to unlock the mutex, as that would mean
			// that the mutex isn't unlocked until all the handlers downstream of this
			// middleware have also returned.
			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

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

		if !user.Activated {
			m.errors.InactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return m.RequireAuthenticatedUser(fn)
}

func (m *Middlewares) RequirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := apicontext.ContextGetUser(r)

		permissions, err := m.models.Permissions.GetAllForUser(r.Context(), user.ID)
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

func (m *Middlewares) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")

		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")

		if origin != "" && len(m.cfg.Cors.TrustedOrigins) != 0 {
			for i := range m.cfg.Cors.TrustedOrigins {
				if origin == m.cfg.Cors.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// Check if the request has the HTTP method OPTIONS and contains the
					// "Access-Control-Request-Method" header. If it does, then we treat
					// it as a preflight request.
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// Set the necessary preflight response headers
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middlewares) Metrics(next http.Handler) http.Handler {
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		totalRequestsReceived.Add(1)

		next.ServeHTTP(w, r)

		// On the way back up the middleware chain, increment the number of responses
		// sent by 1.
		totalResponsesSent.Add(1)

		// Calculate the number of microseconds since we began to process the request,
		// then increment the total processing time by this amount.
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
