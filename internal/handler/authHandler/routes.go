package authhandler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rx3lixir/gateway-service/pkg/token"
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
