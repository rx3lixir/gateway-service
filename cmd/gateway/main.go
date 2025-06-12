package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"
	"github.com/rx3lixir/gateway-service/pkg/health"
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

	// Базовый контекст микросервиса
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Перехват сигналов для graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Configuration",
		"grpc_event_addr", c.Clients.EventClientAddress,
		"grpc_auth_addr", c.Clients.AuthClientAddress,
		"grpc_user_addr", c.Clients.UserClientAddress,
		"http_port", c.Server.HTTPPort,
	)

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

	// Настраиваем health checks
	healthChecker := health.New("event-service", "1.0.0", health.WithTimeout(3*time.Second))

	// Проверяем gRPC соединения
	healthChecker.AddCheck("event-service-connection-check", health.GRPCChecker(eventMcsConn, "event-service"))
	healthChecker.AddCheck("user-service-connection-check", health.GRPCChecker(userMcsConn, "user-service"))
	healthChecker.AddCheck("auth-service-connection-check", health.GRPCChecker(authMcsConn, "auth-service"))

	// Запускаем HTTP сервер для health checks
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", healthChecker.Handler())
	healthMux.HandleFunc("/ready", healthChecker.ReadyHandler())

	// Добавляем liveness probe (просто отвечает 200 OK)
	healthMux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ALIVE"))
	})

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    c.Server.HTTPPort,
		Handler: rootRouter,
	}

	// HTTP Health сервер
	healthServer := &http.Server{
		Addr:    ":8070",
		Handler: healthMux,
	}

	// WaitGroup для координации shutdown
	var wg sync.WaitGroup

	// Запуск HTTP-health сервера в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting health check server", "address", healthServer.Addr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Health check server error", "error", err)
			cancel()
		}
	}()

	// Запуск HTTP сервера gateway в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting HTTP server", "port", c.Server.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	sig := <-sigCh
	log.Info("Received signal, shutting down", "signal", sig)

	// Грэйсфул шатдаун
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Завершение работы основного HTTP сервера
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	}

	// Завершение работы основного HTTP Health сервера
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP Health server shutdown failed", "error", err)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	cancel()

	// Теперь можно безопасно выходить
	log.Info("All servers stopped gracefully")
}
