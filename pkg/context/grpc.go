package context

import (
	"context"
	"net/http"
	"time"
)

// GRPCContextFromHTTP создает gRPC контекст из HTTP запроса с стандартным таймаутом
func GRPCContextFromHTTP(r *http.Request) (context.Context, context.CancelFunc) {
	return GRPCContextFromHTTPWithTimeout(r, 5*time.Second)
}

// GRPCContextFromHTTPWithTimeout создает gRPC контекст с кастомным таймаутом
func GRPCContextFromHTTPWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}

// GRPCContextFromHTTPLongRunning для длительных операций
func GRPCContextFromHTTPLongRunning(r *http.Request) (context.Context, context.CancelFunc) {
	return GRPCContextFromHTTPWithTimeout(r, 30*time.Second)
}
