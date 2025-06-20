package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	contextkeys "github.com/rx3lixir/gateway-service/pkg/context"
)

// AuthMiddleware проверяет JWT токен
func AuthMiddleware(config *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string

			// Сначала пытаемся получить токен из cookie
			if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
				config.Logger.InfoContext(r.Context(), "Using access token from cookie")
			} else {
				// Fallback: получаем из заголовка Authorization
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					config.Logger.WarnContext(r.Context(), "No access token found in cookies or Authorization")
					WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Authorization required"})
					return
				}

				// Проверяем формат заголовка Bearer token
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid authorization header format"})
					return
				}
				tokenString = parts[1]
				config.Logger.InfoContext(r.Context(), "Using access token from Authorization header")
			}

			// Верифицируем токен
			claims, err := config.TokenMaker.VerifyToken(tokenString)
			if err != nil {
				config.Logger.WarnContext(r.Context(), "Invalid token", "error", err)

				// Невалидный токен удаляем из cookies
				if cookie, _ := r.Cookie("access_token"); cookie != nil {
					clearAuthCookies(w)
				}

				WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid or expired token"})
				return
			}

			// Добавляем данные пользователя в контекст запроса
			ctx := context.WithValue(r.Context(), contextkeys.AuthKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// clearAuthCookies очищает все аутентификационные cookies
func clearAuthCookies(w http.ResponseWriter) {
	cookies := []string{"access_token", "refresh_token", "session_id"}
	for _, name := range cookies {
		cookie := &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-time.Hour),
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}
}
