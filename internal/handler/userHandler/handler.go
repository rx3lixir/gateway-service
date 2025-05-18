package userhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"

	contextkeys "github.com/rx3lixir/gateway-service/pkg/contextKeys"
	"github.com/rx3lixir/gateway-service/pkg/password"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

type userHandler struct {
	userClient pbUser.UserServiceClient
	authClient pbAuth.AuthServiceClient
	tokenMaker *token.JWTMaker
	logger     *slog.Logger
}

func NewUserHandler(userClient pbUser.UserServiceClient, authClient pbAuth.AuthServiceClient, secretKey string, log *slog.Logger) *userHandler {
	return &userHandler{
		userClient: userClient,
		authClient: authClient,
		tokenMaker: token.NewJWTMaker(secretKey),
		logger:     log,
	}
}

// createUser создает новго пользователя
func (h *userHandler) createUser(w http.ResponseWriter, r *http.Request) error {
	userReq := new(UserReq)
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode create user request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Валидация входных данных
	if userReq.Email == "" || userReq.Password == "" || userReq.Name == "" {
		return fmt.Errorf("name, email and password are required")
	}

	// Подробное логирование полученных данных
	h.logger.InfoContext(r.Context(), "Received user creation data",
		"username", userReq.Name,
		"email", userReq.Email,
		"isAdmin", userReq.IsAdmin,
	)

	// Хэшируем пароль
	hashed, err := password.Hash(userReq.Password)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to hash user password", "error", err, "user", userReq.Email)
		return fmt.Errorf("failed to hash password: %w", err)
	}
	userReq.Password = hashed

	// Cоздаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Конвертируем данные с клиента в proto
	protoUserReq := toPBUserReq(*userReq)

	createdUser, err := h.userClient.CreateUser(grpcCtx, protoUserReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create user via gRPC",
			"email", createdUser.Email,
			"error", err)
		return err
	}

	// Конвертируем созданного пользователя из proto в ответ
	res := toUserRes(createdUser)

	h.logger.InfoContext(r.Context(), "Created user", "email", userReq.Email, "hashed password", userReq.Password)

	WriteJSON(w, http.StatusOK, res)

	return nil
}

// getUser получает пользователя
func (h *userHandler) getUser(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to get user")

	userId, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Invalid user ID", "error", err)
		return err
	}

	// Создаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Запрашиваем пользователя из user сервиса
	user, err := h.userClient.GetUser(grpcCtx, &pbUser.UserReq{
		Id: userId,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get user via gRPC", "id", userId, "error", err)
		return err
	}

	// Конвертируем пользователя из proto в HTTP ответ
	userRes := toUserRes(user)

	h.logger.InfoContext(r.Context(), "User retrieved successfully", "id", userId)
	return WriteJSON(w, http.StatusOK, userRes)
}

func (h *userHandler) updateUser(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to update user")

	// Извлекаем ID пользователя из URL
	userId, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Invalid user ID", "error", err)
		return err
	}

	// Декодируем запрос на обновление
	updateReq := new(UserReq)
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode update user request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Создаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Подготавливаем запрос на обновление
	protoUserReq := toPBUserReq(*updateReq)
	protoUserReq.Id = userId

	// Обновляем пользователя через gRPC
	updatedUser, err := h.userClient.UpdateUser(grpcCtx, protoUserReq)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to update user via gRPC", "id", userId, "error", err)
		return err
	}

	// Конвертируем обновленного пользователя из proto в HTTP ответ
	userRes := toUserRes(updatedUser)

	h.logger.InfoContext(r.Context(), "User updated successfully", "id", userId)
	return WriteJSON(w, http.StatusOK, userRes)
}

func (h *userHandler) listUsers(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to list users")

	// Получаем claims из HTTP контекста
	claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
	if !ok {
		return fmt.Errorf("unauthorized")
	}

	// Создаем gRPC контекст НА ОСНОВЕ HTTP контекста
	grpcCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Проверка на администратора
	if !claims.IsAdmin {
		h.logger.WarnContext(r.Context(), "Non-admin user tried to list all users", "user_email", claims.Email)
		return fmt.Errorf("permission denied")
	}

	// Запрашиваем у сервиса список пользователей
	users, err := h.userClient.ListUsers(grpcCtx, &pbUser.UserReq{})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to list users via gRPC", "error", err)
		return err
	}

	// Формируем ответ
	listUserRes := new(ListUserRes)
	listUserRes.Users = make([]UserRes, 0, len(users.GetUsers()))

	for _, u := range users.GetUsers() {
		h.logger.InfoContext(r.Context(), "Collecting users...", "user", u.Email)
		listUserRes.Users = append(listUserRes.Users, toUserRes(u))
	}

	h.logger.InfoContext(r.Context(), "Users listed successfully", "count", len(listUserRes.Users))
	return WriteJSON(w, http.StatusOK, listUserRes.Users)
}

func (h *userHandler) deleteUser(w http.ResponseWriter, r *http.Request) error {
	// Извлекаем ID пользователя из URL
	userId, err := parseIDFromURL(r, "id")
	if err != nil {
		h.logger.WarnContext(r.Context(), "Invalid user ID", "error", err)
		return err
	}

	// Cоздаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	_, err = h.userClient.DeleteUser(grpcCtx, &pbUser.UserReq{
		Id: userId,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to delete user via gRPC", "id", userId, "error", err)
		return err
	}

	h.logger.InfoContext(r.Context(), "User deleted successfully", "id", userId)
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
