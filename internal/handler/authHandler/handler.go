package authhandler

import (
	"context"
	"log/slog"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
)

type authHandler struct {
	client pbAuth.AuthServiceClient
	logger *slog.Logger
}

func NewEventHandler(client pbAuth.AuthServiceClient, ctx context.Context, log *slog.Logger) *authHandler {
	return &authHandler{
		client: client,
		logger: log,
	}
}
