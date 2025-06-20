package userhandler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rx3lixir/gateway-service/pkg/middleware"
)

func RegisterRoutes(u *userHandler) *chi.Mux {
	r := chi.NewRouter()

	middlewareConfig := &middleware.Config{
		TokenMaker: u.tokenMaker,
		Logger:     u.logger,
		CORSConfig: middleware.DefaultCORSConfig(),
	}

	// Общие middleware
	for _, mw := range middleware.CommonMiddlewares(middlewareConfig) {
		r.Use(mw)
	}

	// API routes
	r.Route("/api/v1/users", func(r chi.Router) {
		// Публичные эндпоинты (регистрация)
		r.Post("/", u.makeHTTPHandlerFunc(u.createUser))

		// Защищенные эндпоинты (нужна аутентификация)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(middlewareConfig, false))
			r.Get("/{id}", u.makeHTTPHandlerFunc(u.getUser))
			r.Put("/{id}", u.makeHTTPHandlerFunc(u.updateUser))
		})

		// Защищенные эндпоинты (нужны права админа)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(middlewareConfig, true))
			r.Get("/", u.makeHTTPHandlerFunc(u.listUsers))
			r.Delete("/{id}", u.makeHTTPHandlerFunc(u.deleteUser))
		})
	})

	return r
}
