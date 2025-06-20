package authhandler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rx3lixir/gateway-service/pkg/middleware"
)

func RegisterRoutes(a *authHandler) *chi.Mux {
	r := chi.NewRouter()

	// Конфиг middleware
	middlewareConfig := &middleware.Config{
		TokenMaker: a.tokenMaker,
		Logger:     a.logger,
		CORSConfig: middleware.DefaultCORSConfig(),
	}

	// Применяем общие middleware
	for _, mw := range middleware.CommonMiddlewares(middlewareConfig) {
		r.Use(mw)
	}

	r.Use(middleware.SecurityHeaders())

	// API routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		// Публичные эндпоинты (не нужна аутентификация)
		r.Post("/login", a.makeHTTPHandlerFunc(a.handleLogin))
		r.Post("/refresh", a.makeHTTPHandlerFunc(a.handleRefreshToken))

		// Защищенные эндпоинты (нужна аутентификация)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(middlewareConfig, false))

			r.Get("/me", a.makeHTTPHandlerFunc(a.handleMe))
			r.Post("/logout", a.makeHTTPHandlerFunc(a.handleLogout))
			r.Post("/revoke", a.makeHTTPHandlerFunc(a.handleRevokeToken))
		})
	})

	return r
}
