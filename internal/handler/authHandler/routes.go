package authhandler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rx3lixir/gateway-service/internal/token"
)

// Структура для передачи в контекст
type authContextKey struct{}

func RegisterRoutes(a *authHandler) *chi.Mux {
	r := chi.NewRouter()

	// Общие middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// API routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		// Публичные эндпоинты (не нужна аутентификация)
		r.Post("/login", a.makeHTTPHandlerFunc(a.handleLogin))
		r.Post("/refresh", a.makeHTTPHandlerFunc(a.handleRefreshToken))

		// Защищенные эндпоинты (нужна аутентификация)
		r.Group(func(r chi.Router) {
			r.Use(a.authMiddleware)
			r.Post("/logout", a.makeHTTPHandlerFunc(a.handleLogout))
			r.Post("/revoke", a.makeHTTPHandlerFunc(a.handleRevokeToken))
		})
	})

	return r
}

// authMiddleware проверяет токен в заголовке Authorization
func (h *authHandler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Authorization header is required"})
			return
		}

		// Проверяем формат заголовка Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]

		// Верифицируем токен
		claims, err := h.tokenMaker.VerifyToken(tokenString)
		if err != nil {
			h.logger.WarnContext(r.Context(), "Invalid token", "error", err)
			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid or expired token"})
			return
		}

		// Добавляем данные пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), authContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddleware проверяет, является ли пользователь администратором
func (h *authHandler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем данные пользователя из контекста
		claims, ok := r.Context().Value(authContextKey{}).(*token.UserClaims)
		if !ok || claims == nil {
			h.logger.WarnContext(r.Context(), "No auth claims found in context")
			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Unauthorized"})
			return
		}

		// Проверяем права администратора
		if !claims.IsAdmin {
			h.logger.WarnContext(r.Context(), "Access denied: user is not admin", "email", claims.Email)
			WriteJSON(w, http.StatusForbidden, APIError{Error: "Access denied: admin privileges required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start запускает HTTP сервер
func Start(addr string, router *chi.Mux) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}

// GracefulShutdown выполняет корректное завершение работы сервера
func GracefulShutdown(ctx context.Context, server *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to gracefully shutdown server", "error", err)
	}
}
