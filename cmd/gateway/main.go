package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"

	"github.com/rx3lixir/gateway-service/internal/handler/authHandler"
	"github.com/rx3lixir/gateway-service/internal/handler/eventHandler"

	"github.com/ianschenck/envflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		grpc_EventSvc_addr = envflag.String("GRPC_EVENTSERVICE_ADDR", "0.0.0.0:9091", "!")
		httpPort           = envflag.String("HTTP_PORT", ":8080", "HTTP server port")
	)
	envflag.Parse()

	// Подключаем логирование
	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})

	logger := slog.New(slogHandler)
	logger.Info("Starting gateway service", "version", "1.0.0")
	logger.Info("Configuration", "grpc_event_addr", *grpc_EventSvc_addr, "http_port", *httpPort)

	// Базовый контекст микросервиса
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	logger.Info("Connecting to gRPC event service", "address", *grpc_EventSvc_addr)

	// Соединяемся с ивент-сервисом
	conn, err := grpc.NewClient(*grpc_EventSvc_addr, opts...)
	if err != nil {
		logger.Error("Failed to connect to event service", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	logger.Info("Connected to gRPC event service")

	// Создание gRPC клиента
	eventClient := pbEvent.NewEventServiceClient(conn)
	authClient := pbAuth.NewAuthServiceClient(conn)

	// Создание обработчика событий
	eHandler := eventHandler.NewEventHandler(eventClient, ctx, logger)
	aHandler := authhandler.NewAuthHandler(authClient, ctx, logger)

	// Регистрация маршрутов
	eventRoutes := eventHandler.RegisterRoutes(eHandler)

	// Перехват сигналов для graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		logger.Info("Starting HTTP server", "port", *httpPort)
		if err := eventHandler.Start(*httpPort, eventRoutes); err != nil {
			logger.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	sig := <-sigCh
	logger.Info("Received signal, shutting down", "signal", sig)
	cancel()
}
