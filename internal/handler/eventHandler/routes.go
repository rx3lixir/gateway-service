package eventHandler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rx3lixir/gateway-service/pkg/middleware"
)

func RegisterRoutes(e *eventHandler) *chi.Mux {
	r := chi.NewRouter()

	// Конфигурация middleware
	middlewareConfig := &middleware.Config{
		TokenMaker: e.tokenMaker,
		Logger:     e.logger,
		CORSConfig: middleware.DefaultCORSConfig(),
	}

	// Общие middleware
	for _, mw := range middleware.CommonMiddlewares(middlewareConfig) {
		r.Use(mw)
	}

	r.Route("/api/v1", func(r chi.Router) {
		// События: без аутентификации
		r.Get("/events", e.makeHTTPHandlerFunc(e.handleGetEvents))
		r.Get("/events/{id}", e.makeHTTPHandlerFunc(e.handleGetEventByID))

		// Поиск : без аутентификации
		r.Post("/events/search", e.makeHTTPHandlerFunc(e.handleGetEventsAdvanced))
		r.Get("/events/suggestions", e.makeHTTPHandlerFunc(e.handleGetSuggestions))

		// Категории: без аутентификации
		r.Get("/categories", e.makeHTTPHandlerFunc(e.handleListCategories))
		r.Get("/categories/{id}", e.makeHTTPHandlerFunc(e.handleGetCategoryByID))

		// Защищенные эндпоинты
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(middlewareConfig, true))

			// Админские операции для событий
			r.Delete("/events/{id}", e.makeHTTPHandlerFunc(e.handleDeleteEvent))
			r.Patch("/events/{id}", e.makeHTTPHandlerFunc(e.handleUpdateEvent))
			r.Post("/events", e.makeHTTPHandlerFunc(e.handleCreateEvent))

			// Админские операции для категорий
			r.Post("/categories", e.makeHTTPHandlerFunc(e.handleCreateCategory))
			r.Patch("/categories/{id}", e.makeHTTPHandlerFunc(e.handleUpdateCategory))
			r.Delete("/categories/{id}", e.makeHTTPHandlerFunc(e.handleDeleteCategory))
		})
	})

	return r
}
