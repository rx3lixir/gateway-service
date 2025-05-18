package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"

	"github.com/rx3lixir/gateway-service/internal/handler/authHandler"
	"github.com/rx3lixir/gateway-service/internal/handler/eventHandler"
	"github.com/rx3lixir/gateway-service/internal/handler/userHandler"

	"github.com/ianschenck/envflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		grpc_eventSvc_addr = envflag.String("GRPC_EVENTSERVICE_ADDR", "0.0.0.0:9091", "Event service gRPC address")
		grpc_authSvc_addr  = envflag.String("GRPC_AUTHSERVICE_ADDR", "0.0.0.0:9092", "Auth service gRPC address")
		grpc_userSvc_addr  = envflag.String("GRPC_USERSERVICE_ADDR", "0.0.0.0:9093", "User service gRPC address")
		httpPort           = envflag.String("HTTP_PORT", ":8080", "HTTP server port")
		secretKey          = envflag.String("SECRET_KEY", "36080001349340267925113477454910", "For JWT encoding")
	)
	envflag.Parse()

	// Подключаем логирование
	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})

	logger := slog.New(slogHandler)

	logger.Info("Starting gateway service", "version", "1.0.0")
	logger.Info("Configuration",
		"grpc_event_addr", *grpc_eventSvc_addr,
		"grpc_auth_addr", *grpc_authSvc_addr,
		"grpc_user_addr", *grpc_userSvc_addr,
		"http_port", *httpPort)

	// Базовый контекст микросервиса
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Соединяемся с сервисом авторизации
	authMcsConn, err := grpc.NewClient(*grpc_authSvc_addr, opts...)
	if err != nil {
		logger.Error("Failed to connect to auth service", "error", err)
		os.Exit(1)
	}
	defer authMcsConn.Close()
	logger.Info("Connected to gRPC auth service")

	// Соединяемся с сервисом пользователей
	userMcsConn, err := grpc.NewClient(*grpc_userSvc_addr, opts...)
	if err != nil {
		logger.Error("Failed to connect to user service", "error", err)
		os.Exit(1)
	}
	defer userMcsConn.Close()
	logger.Info("Connected to gRPC user service")

	// Соединяемся с сервисом событий
	eventMcsConn, err := grpc.NewClient(*grpc_eventSvc_addr, opts...)
	if err != nil {
		logger.Error("Failed to connect to event service", "error", err)
		os.Exit(1)
	}
	defer eventMcsConn.Close()
	logger.Info("Connected to gRPC event service")

	// Создание gRPC клиентов
	eventClient := pbEvent.NewEventServiceClient(eventMcsConn)
	authClient := pbAuth.NewAuthServiceClient(authMcsConn)
	userClient := pbUser.NewUserServiceClient(userMcsConn)

	// Создание обработчиков
	eHandler := eventHandler.NewEventHandler(eventClient, *secretKey, logger)
	aHandler := authhandler.NewAuthHandler(authClient, userClient, *secretKey, logger)
	uHandler := userhandler.NewUserHandler(userClient, authClient, *secretKey, logger)

	// Регистрация маршрутов
	eventRoutes := eventHandler.RegisterRoutes(eHandler)
	authRoutes := authhandler.RegisterRoutes(aHandler)
	userRoutes := userhandler.RegisterRoutes(uHandler)

	// Создаем корневой роутер для объединения маршрутов
	rootRouter := chi.NewRouter()

	// Монтируем роутеры на корневой роутер
	rootRouter.Mount("/event", eventRoutes)
	rootRouter.Mount("/auth", authRoutes)
	rootRouter.Mount("/user", userRoutes)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    *httpPort,
		Handler: rootRouter,
	}

	// Перехват сигналов для graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		logger.Info("Starting HTTP server", "port", *httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	sig := <-sigCh
	logger.Info("Received signal, shutting down", "signal", sig)
	cancel()

	// Грэйсфул шатдаун
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
	}

	cancel()
}
