package userhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"

	"github.com/rx3lixir/gateway-service/pkg/password"
	"github.com/rx3lixir/gateway-service/pkg/token"
)

type userHandler struct {
	userClient pbUser.UserServiceClient
	authClient pbAuth.AuthServiceClient
	tokenMaker *token.JWTMaker
	logger     *slog.Logger
}

func NewEventHandler(userClient pbUser.UserServiceClient, secretKey string, log *slog.Logger) *userHandler {
	return &userHandler{
		userClient: userClient,
		tokenMaker: token.NewJWTMaker(secretKey),
		logger:     log,
	}
}

func (h *userHandler) createUser(w http.ResponseWriter, r *http.Request) error {
	userReq := new(UserReq)
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode create user request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}

	// Подробное логирование полученных данных
	h.logger.InfoContext(r.Context(), "Received user creation data",
		"username", userReq.Name,
		"email", userReq.Email,
		"password", userReq.Password,
		"isAdmin", userReq.IsAdmin,
	)

	// Хэшируем пароль
	hashed, err := password.Hash(userReq.Password)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to hash user password", "error", err, "user", userReq.Email)
		return fmt.Errorf("failed to hash password", err)
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

	WriteJSON(w, http.StatusOK, res)

	return nil
}

func (h *userHandler) listUsers(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling request to list users")

	// Cоздаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	users, err := h.userClient.ListUsers(grpcCtx, &pbUser.UserReq{})
	if err != nil {
		h.logger.WarnContext(r.Context(), "Error processing ListUsers request", "error", err)
		return fmt.Errorf("Error calling user client via gRPC method ListUsers", err)
	}
	h.logger.InfoContext(r.Context(), "Requesting all users")

	listUserRes := new(ListUserRes)

	for _, u := range users.GetUsers() {
		h.logger.InfoContext(r.Context(), "Collecting users...", "user", u.Email)
		listUserRes.Users = append(listUserRes.Users, toUserRes(u))
	}

	WriteJSON(w, http.StatusOK, listUserRes.Users)

	return nil
}

func (h *userHandler) deleteUser(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	i, err := parseInt64(id)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Error parsing id to int64", "id", id)
		return fmt.Errorf("Error parsing id", err)
	}

	// Cоздаем gRPC контекст для запроса
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	_, err = h.userClient.DeleteUser(grpcCtx, &pbUser.UserReq{
		Id: i,
	})
	if err != nil {
		h.logger.WarnContext(r.Context(), "Error getting user", "id", id)
		return fmt.Errorf("User deleting error", err)
	}

	WriteJSON(w, http.StatusOK, "User deleted")

	return nil
}
