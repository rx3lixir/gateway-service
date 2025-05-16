package eventHandler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func RegisterRoutes(e *eventHandler) *chi.Mux {
	r := chi.NewRouter()

	// middleware для логирования и восстановления от паники
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/events", e.makeHTTPHandlerFunc(e.handleGetEvents))
		r.Post("/events", e.makeHTTPHandlerFunc(e.handleCreateEvent))
		r.Get("/events/{id}", e.makeHTTPHandlerFunc(e.handleGetEventByID))
		r.Put("/events/{id}", e.makeHTTPHandlerFunc(e.handleUpdateEvent))
		r.Delete("/events/{id}", e.makeHTTPHandlerFunc(e.handleDeleteEvent))
	})

	return r
}

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

func GracefulShutdown(ctx context.Context, server *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to graceful shutdown server", "error", err)
	}
}
