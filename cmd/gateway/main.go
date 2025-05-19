package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"
	"github.com/rx3lixir/gateway-service/pkg/logger"

	"github.com/rx3lixir/gateway-service/internal/config"
	"github.com/rx3lixir/gateway-service/internal/handler/authHandler"
	"github.com/rx3lixir/gateway-service/internal/handler/eventHandler"
	"github.com/rx3lixir/gateway-service/internal/handler/userHandler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	c, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера
	logger.Init(c.Service.Env)
	defer logger.Close()

	// Создаем экземпляр логгера для передачи компонентам
	log := logger.NewLogger()

	log.Info("Starting gateway service", "version", "1.0.0")
	log.Info("Configuration",
		"grpc_event_addr", c.Clients.EventClientAddress,
		"grpc_auth_addr", c.Clients.AuthClientAddress,
		"grpc_user_addr", c.Clients.UserClientAddress,
		"http_port", c.Server.HTTPPort)

	// Базовый контекст микросервиса
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Соединяемся с сервисом авторизации
	authMcsConn, err := grpc.NewClient(c.Clients.AuthClientAddress, opts...)
	if err != nil {
		log.Error("Failed to connect to auth service", "error", err)
		os.Exit(1)
	}
	defer authMcsConn.Close()
	log.Info("Connected to gRPC auth service")

	// Соединяемся с сервисом пользователей
	userMcsConn, err := grpc.NewClient(c.Clients.UserClientAddress, opts...)
	if err != nil {
		log.Error("Failed to connect to user service", "error", err)
		os.Exit(1)
	}
	defer userMcsConn.Close()
	log.Info("Connected to gRPC user service")

	// Соединяемся с сервисом событий
	eventMcsConn, err := grpc.NewClient(c.Clients.EventClientAddress, opts...)
	if err != nil {
		log.Error("Failed to connect to event service", "error", err)
		os.Exit(1)
	}
	defer eventMcsConn.Close()
	log.Info("Connected to gRPC event service")

	// Создание gRPC клиентов
	eventClient := pbEvent.NewEventServiceClient(eventMcsConn)
	authClient := pbAuth.NewAuthServiceClient(authMcsConn)
	userClient := pbUser.NewUserServiceClient(userMcsConn)

	// Создание обработчиков
	eHandler := eventHandler.NewEventHandler(eventClient, c.Service.SecretKey, log)
	aHandler := authhandler.NewAuthHandler(authClient, userClient, c.Service.SecretKey, log)
	uHandler := userhandler.NewUserHandler(userClient, authClient, c.Service.SecretKey, log)

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
		Addr:    c.Server.HTTPPort,
		Handler: rootRouter,
	}

	// Перехват сигналов для graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		log.Info("Starting HTTP server", "port", c.Server.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	sig := <-sigCh
	log.Info("Received signal, shutting down", "signal", sig)
	cancel()

	// Грэйсфул шатдаун
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	}

	cancel()
}
