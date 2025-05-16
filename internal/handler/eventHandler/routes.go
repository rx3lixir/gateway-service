package eventHandler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var r *chi.Mux

func RegisterRoutes(e *eventHandler) *chi.Mux {
	r = chi.NewRouter()

	r.Get("/events", e.makeHTTPHandlerFunc(e.handleGetEvents))
	r.Post("/events", e.makeHTTPHandlerFunc(e.handleCreateEvent))
	r.Get("/events/{id}", e.makeHTTPHandlerFunc(e.handleGetEventByID))
	r.Put("/events/{id}", e.makeHTTPHandlerFunc(e.handleUpdateEvent))
	r.Delete("/events/{id}", e.makeHTTPHandlerFunc(e.handleDeleteEvent))

	return r
}

func Start(addr string) error {
	return http.ListenAndServe(addr, r)
}
