package handlers

import (
	"context"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
)

type EventHandler struct {
	ctx    context.Context
	client pbEvent.EventServiceClient
}

func NewEventHandler(client pbEvent.EventServiceClient, ctx context.Context) *EventHandler {
	return &EventHandler{
		ctx:    ctx,
		client: client,
	}
}
