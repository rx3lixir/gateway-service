package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ianschenck/envflag"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	"github.com/rx3lixir/gateway-service/internal/handler/eventHandler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		grpc_EventSvc_addr = envflag.String("GRPC_EVENTSERVICE_ADDR", "0.0.0.0:9091", "!")
	)

	slogHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	logger := slog.New(slogHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(*grpc_EventSvc_addr, opts...)
	if err != nil {
		slog.Error("failed to connect to server", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pbEvent.NewEventServiceClient(conn)
	pbEvent.NewEventServiceClient(conn)

	hdl := eventHandler.NewEventHandler(client, ctx, logger)

	eventHandler.RegisterRoutes(hdl)

	eventHandler.Start(":8080")

	slog.Info("gRPC client started", "port", ":8080")
}
