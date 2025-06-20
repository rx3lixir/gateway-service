package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// CommonMiddlewares возвращает набор общих middleware
func CommonMiddlewares(config *Config) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(30 * time.Second),
		CORSMiddleware(config.CORSConfig),
		middleware.Compress(5), // gzip compression
		middleware.RequestID,   // добавляет request ID
	}
}

// SecurityHeaders добавляет security заголовки
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware простое rate limiting (можно расширить)
func RateLimitMiddleware() func(http.Handler) http.Handler {
	return middleware.Throttle(100) // 100 concurrent requests
}
