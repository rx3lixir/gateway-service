package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/rx3lixir/gateway-service/pkg/logger"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

// APIError представляет структуру ошибки для ответов API
type APIError struct {
	Error string `json:"error"`
}

// Config конфигурация для middleware
type Config struct {
	TokenMaker *token.JWTMaker
	Logger     logger.Logger
	CORSConfig CORSConfig
}

// CORSConfig конфигурация CORS
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// WriteJSON отправляет JSON ответ
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data == nil && (statusCode == http.StatusNoContent || statusCode == http.StatusAccepted) {
		return nil
	}

	if data == nil {
		data = map[string]any{}
	}

	return json.NewEncoder(w).Encode(data)
}

// DefaultCORSConfig возвращает дефолтную CORS конфигурацию
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}
}
