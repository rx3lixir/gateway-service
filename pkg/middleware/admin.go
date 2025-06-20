package middleware

import (
	"net/http"

	contextkeys "github.com/rx3lixir/gateway-service/pkg/context"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

// AdminMiddleware проверяет права администратора
func AdminMiddleware(config *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
			if !ok || claims == nil {
				config.Logger.WarnContext(r.Context(), "No auth claims found in context")
				WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Unauthorized"})
				return
			}

			if !claims.IsAdmin {
				config.Logger.WarnContext(r.Context(), "Access denied: user is not admin", "email", claims.Email)
				WriteJSON(w, http.StatusForbidden, APIError{Error: "Access denied: admin privileges required"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth middleware что требует аутентификации (комбинация auth + admin при необходимости)
func RequireAuth(config *Config, requireAdmin bool) func(http.Handler) http.Handler {
	if requireAdmin {
		return func(next http.Handler) http.Handler {
			return AuthMiddleware(config)(AdminMiddleware(config)(next))
		}
	}
	return AuthMiddleware(config)
}
