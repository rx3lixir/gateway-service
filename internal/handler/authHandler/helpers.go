package authhandler

import (
	"google.golang.org/grpc/codes"  // Для кодов gRPC ошибо
	"google.golang.org/grpc/status" // Для обработки gRPC ошибок

	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// APIError представляет структуру ошибки для ответов API.
type APIError struct {
	Error string `json:"error"`
}

// WriteJSON отправляет данные в формате JSON с указанным HTTP статусом.
// Автоматически устанавливает правильный Content-Type заголовок.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data == nil && (statusCode == http.StatusNoContent || statusCode == http.StatusAccepted) {
		return nil
	}
	if data == nil && statusCode != http.StatusNoContent && statusCode != http.StatusAccepted {
		// Для других статусов, если data nil, возвращаем пустой JSON объект или массив в зависимости от ожиданий
		// Если ожидается объект:
		data = map[string]interface{}{}
		// Если ожидается массив (например, для списков, которые могут быть пустыми):
		// if _, ok := data.([]interface{}); ok || (data == nil && strings.Contains(r.URL.Path, "events")) { // Простой пример для определения, когда нужен массив
		// data = []interface{}{}
		// }
	}
	return json.NewEncoder(w).Encode(data)
}

// apiFunc определяет сигнатуру функций-обработчиков API,
// которые возвращают ошибку для централизованной обработки.
type apiFunc func(w http.ResponseWriter, r *http.Request) error

// makeHTTPHandleFunc преобразует apiFunc в стандартный http.HandlerFunc,
// добавляя унифицированную обработку ошибок.
func (h *authHandler) makeHTTPHandlerFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			// Обработка gRPC ошибок
			st, ok := status.FromError(err)
			if ok {
				var httpStatus int
				switch st.Code() {
				case codes.NotFound:
					httpStatus = http.StatusNotFound
				case codes.InvalidArgument:
					httpStatus = http.StatusBadRequest
				case codes.AlreadyExists:
					httpStatus = http.StatusConflict
				case codes.Unauthenticated:
					httpStatus = http.StatusUnauthorized
				case codes.PermissionDenied:
					httpStatus = http.StatusForbidden
				// Добавьте другие коды gRPC по мере необходимости
				default:
					h.logger.Error("Unhandled gRPC error", "code", st.Code(), "message", st.Message(), "path", r.URL.Path)
					httpStatus = http.StatusInternalServerError
				}
				WriteJSON(w, httpStatus, APIError{Error: st.Message()})
				return
			}

			// Обработка "обычных" ошибок приложения (например, парсинг ID, JSON)
			// Эти ошибки обычно должны приводить к http.StatusBadRequest или http.StatusInternalServerError
			// Проверка на "is required" или "invalid"
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "required") || strings.Contains(errStr, "invalid") || strings.Contains(errStr, "format") || strings.Contains(errStr, "positive integer") {
				WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
				return
			}

			if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "unauthenticated") {
				WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Unauthorized"})
				return
			}

			// Если ошибка содержит "not found" (из старого кода, но лучше полагаться на gRPC codes.NotFound)
			if strings.Contains(errStr, "not found") {
				WriteJSON(w, http.StatusNotFound, APIError{Error: err.Error()})
				return
			}

			h.logger.Error("HTTP handler error", "error", err, "path", r.URL.Path)
			WriteJSON(w, http.StatusInternalServerError, APIError{Error: "An unexpected error occurred"})
		}
	}
}

// createContext создает дочерний контекст с таймаутом для gRPC вызова.
func (h *authHandler) createContext(r *http.Request) (context.Context, context.CancelFunc) {
	// Установите подходящий таймаут для ваших gRPC вызовов
	return context.WithTimeout(r.Context(), 5*time.Second)
}
