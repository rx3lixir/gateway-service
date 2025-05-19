package userhandler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	contextkeys "github.com/rx3lixir/gateway-service/pkg/contextKeys"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

func RegisterRoutes(u *userHandler) *chi.Mux {
	r := chi.NewRouter()

	// Общие middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Добавляем CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // <- Разрешаем фронту доступ
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true, // Ecли будут куки или токены
	}))

	// API routes
	r.Route("/api/v1/users", func(r chi.Router) {
		// Публичные эндпоинты (регистрация)
		r.Post("/", u.makeHTTPHandlerFunc(u.createUser))

		// Защищенные эндпоинты (нужна аутентификация)
		r.Group(func(r chi.Router) {
			r.Use(u.authMiddleware)
			r.Get("/{id}", u.makeHTTPHandlerFunc(u.getUser))
			r.Put("/{id}", u.makeHTTPHandlerFunc(u.updateUser))
		})

		// Защищенные эндпоинты (нужны права админа)
		r.Group(func(r chi.Router) {
			r.Use(u.authMiddleware)
			r.Use(u.adminMiddleware)
			r.Get("/", u.makeHTTPHandlerFunc(u.listUsers))
			r.Delete("/{id}", u.makeHTTPHandlerFunc(u.deleteUser))
		})
	})

	return r
}

// authMiddleware проверяет токен в заголовке Authorization
func (h *userHandler) authMiddleware(next http.Handler) http.Handler {
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
		ctx := context.WithValue(r.Context(), contextkeys.AuthKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddleware проверяет, является ли пользователь администратором
func (h *userHandler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем данные пользователя из контекста
		claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
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
