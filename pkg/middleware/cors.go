package middleware

import (
	"github.com/go-chi/cors"
	"net/http"
)

// CORSMiddleware обрабатывает CORS запросы
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           86400,
	})
}
