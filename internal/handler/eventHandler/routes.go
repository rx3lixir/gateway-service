package eventHandler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	contextkeys "github.com/rx3lixir/gateway-service/pkg/contextKeys"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

func RegisterRoutes(e *eventHandler) *chi.Mux {
	r := chi.NewRouter()

	// Общие middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		// События: без аутентификации
		r.Get("/events", e.makeHTTPHandlerFunc(e.handleGetEvents))
		r.Get("/events/{id}", e.makeHTTPHandlerFunc(e.handleGetEventByID))
		r.Post("/events/search", e.makeHTTPHandlerFunc(e.handleGetEventsAdvanced))

		// Категории: без аутентификации
		r.Get("/categories", e.makeHTTPHandlerFunc(e.handleListCategories))
		r.Get("/categories/{id}", e.makeHTTPHandlerFunc(e.handleGetCategoryByID))

		// Защищенные эндпоинты
		r.Group(func(r chi.Router) {
			r.Use(e.authMiddleware)
			r.Use(e.adminMiddleware)

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

// authMiddleware проверяет токен в заголовке Authorization
func (h *eventHandler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// Cначала пытаемся получить токен из cookie
		if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
			tokenString = cookie.Value
			h.logger.InfoContext(r.Context(), "Using access token from cookie")
		} else {
			// Fallback: получаем из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				h.logger.WarnContext(r.Context(), "No access token found in cookies or Authorizaion")
				WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid authorization header format"})
				return
			}
			// Проверяем формат заголовка Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid authorization header format"})
				return
			}
			tokenString = parts[1]
			h.logger.InfoContext(r.Context(), "Using access token from Authorization header")
		}

		// Верифицируем токен
		claims, err := h.tokenMaker.VerifyToken(tokenString)
		if err != nil {
			h.logger.WarnContext(r.Context(), "Invalid token", "error", err)

			// Невалидный токен удаляем из cookies
			if cookie, _ := r.Cookie("access_token"); cookie != nil {
				h.clearCookies(w)
			}

			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Invalid or expired token"})
			return
		}

		// Добавляем данные пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), contextkeys.AuthKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminMiddleware проверяет права администратора
func (h *eventHandler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
		if !ok || claims == nil {
			h.logger.WarnContext(r.Context(), "No auth claims found in context")
			WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Unauthorized"})
			return
		}

		if !claims.IsAdmin {
			h.logger.WarnContext(r.Context(), "Access denied: user is not admin", "email", claims.Email)
			WriteJSON(w, http.StatusForbidden, APIError{Error: "Access denied: admin privileges required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
