package middleware

import (
	"admin-stats/server/logger"
	"net/http"
)

// UserAuth validates a Bearer JWT from the Authorization header.
// TODO: validate token, extract claims, set in request context.
func UserAuth(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				log.Warn("missing authorization header")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// InternalAuth validates a static token from the X-Internal-Key header.
// The expected token is loaded from config at startup as we want static authentication for given assignment.
// In a real-world scenario, we would use a more secure approach, such as JWTs or OAuth2, for internal authentication.
func InternalAuth(log logger.Logger, expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Internal-Key")
			if key == "" || key != expectedToken {
				log.Warn("invalid or missing internal key")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
