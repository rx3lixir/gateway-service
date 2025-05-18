package userhandler

import (
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"  // Для кодов gRPC ошибо
	"google.golang.org/grpc/status" // Для обработки gRPC ошибок

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError представляет структуру ошибки для ответов API.
type APIError struct {
	Error string `json:"error"`
}

// parseInt64 преобразует строку в int64 для использования в запросах
func parseInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
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
func (h *userHandler) makeHTTPHandlerFunc(f apiFunc) http.HandlerFunc {
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

// parseIDFromURL извлекает и валидирует ID из URL. Изменен на int64.
func parseIDFromURL(r *http.Request, paramName string) (int64, error) {
	idParam := chi.URLParam(r, paramName)
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s format: %v", paramName, idParam)
	}
	if id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %d", paramName, id)
	}
	return id, nil
}

// createContext создает дочерний контекст с таймаутом для gRPC вызова.
func (h *userHandler) createContext(r *http.Request) (context.Context, context.CancelFunc) {
	// Установите подходящий таймаут для ваших gRPC вызовов
	return context.WithTimeout(r.Context(), 5*time.Second)
}
