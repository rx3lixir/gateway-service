package authhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"

	"github.com/rx3lixir/gateway-service/internal/token"
	"github.com/rx3lixir/gateway-service/pkg/password"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authHandler struct {
	authClient pbAuth.AuthServiceClient
	userClient pbUser.UserServiceClient
	tokenMaker *token.JWTMaker
	logger     *slog.Logger
	ctx        context.Context
}

func NewAuthHandler(client pbAuth.AuthServiceClient, ctx context.Context, log *slog.Logger) *authHandler {
	return &authHandler{
		authClient: client,
		logger:     log,
	}
}

// handleLogin обрабатывает запрос на логин пользователя
func (h *authHandler) handleLogin(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling login request")

	loginRequest := new(LoginUserReq)
	if err := json.NewDecoder(r.Body).Decode(loginRequest); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode login request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Валидация входных данных
	if strings.TrimSpace(loginRequest.Email) == "" || strings.TrimSpace(loginRequest.Password) == "" {
		return fmt.Errorf("email and password are required")
	}

	// Создаем gRPC конетекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Получаем пользователя по email из User сервиса
	user, err := h.userClient.GetUser(grpcCtx, &pbUser.UserReq{
		Email: loginRequest.Email,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get user", "email", loginRequest.Email, "error", err)

		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return fmt.Errorf("user not found with email: %s", loginRequest.Email)
		}
		return err
	}

	ok := password.Verify(loginRequest.Password, user.Password)
	if !ok {
		h.logger.WarnContext(grpcCtx, "Invalid password attempt", "email", loginRequest.Email)
		return fmt.Errorf("invalid credentials")
	}

	// Создаем токены доступа
	accessToken, acessClaims, err := h.tokenMaker.CreateToken(
		int(user.Id),
		user.Email,
		user.IsAdmin,
		time.Minute*15,
	)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create access token", "error", err)
		return fmt.Errorf("error creating token: %w", err)
	}

	refreshToken, refreshClaims, err := h.tokenMaker.CreateToken(
		int(user.Id),
		user.Email,
		user.IsAdmin,
		time.Hour*24,
	)

	// Сохраняем сессию в Auth сервисе
	session, err := h.authClient.CreateSession(grpcCtx, &pbAuth.SessionReq{
		Id:           refreshClaims.RegisteredClaims.ID,
		UserEmail:    user.Email,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    nil, // Заполнится в auth сервисе
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create session", "error", err)
		return fmt.Errorf("error creating session: %w", err)
	}

	// Формируем ответ
	response := LoginUserRes{
		SessionID:            session.Id,
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: acessClaims.RegisteredClaims.ExpiresAt.Time,
		User: UserRes{
			Name:    user.Name,
			Email:   user.Email,
			IsAdmin: user.IsAdmin,
		},
	}

	h.logger.InfoContext(grpcCtx, "User logged in successfully", "email", user.Email)
	return WriteJSON(w, http.StatusOK, response)
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling logout request")

	// Получаем данные пользователя из контекста
	claims, ok := r.Context().Value(authContextKey{}).(*token.UserClaims)
	if !ok || claims == nil {
		h.logger.WarnContext(r.Context(), "No auth claims found in context")
		return fmt.Errorf("unauthorized")
	}

	// создаем gRPC контекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Удаляем сессию из Auth сервиса
	_, err := h.authClient.DeleteSession(grpcCtx, &pbAuth.SessionReq{
		Id: claims.RegisteredClaims.ID,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to delete session", "session_id", claims.RegisteredClaims.ID, "error", err)
		return fmt.Errorf("error deleting session: %w", err)
	}

	h.logger.InfoContext(grpcCtx, "User logged out successfully", "email", claims.Email)
	return WriteJSON(w, http.StatusNoContent, nil)
}

// handleRefreshToken обрабатывает запрос на обновление токена доступа
func (h *authHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling token refresh request")

	req := new(RenewAccessTokenReq)

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode refresh token request", "error", err)
		return fmt.Errorf("invalid request body: %w", err)
	}
	defer r.Body.Close()

	// Валидируем refresh token
	refreshClaims, err := h.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Invalid refresh token", "error", err)
		return fmt.Errorf("invalid refresh token: %w", err)
	}

	// Создаем gRPC контекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Проверяем статус сессии
	session, err := h.authClient.GetSession(grpcCtx, &pbAuth.SessionReq{
		Id: refreshClaims.RegisteredClaims.ID,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get session", "session_id", refreshClaims.RegisteredClaims.ID, "error", err)
		return fmt.Errorf("error getting session: %w", err)
	}

	// Проверяем что сессия не отозвана
	if session.IsRevoked {
		h.logger.WarnContext(grpcCtx, "Session is revoked", "session_id", session.Id)
		return fmt.Errorf("session is revoked")
	}

	// Сверяем email из сессии с email в токене
	if session.UserEmail != refreshClaims.Email {
		h.logger.WarnContext(grpcCtx, "Session email mismatch",
			"token_email", refreshClaims.Email,
			"session_email", session.UserEmail)
		return fmt.Errorf("invalid session")
	}

	// Создаем новый acess token
	accessToken, accessClaims, err := h.tokenMaker.CreateToken(
		refreshClaims.Id,
		refreshClaims.Email,
		refreshClaims.IsAdmin,
		time.Minute*15,
	)
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to create new acess token", "error", err)
		return fmt.Errorf("error creating token: %w", err)
	}

	// Формируем ответ с новым токеном
	response := RenewAccessTokenRes{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessClaims.RegisteredClaims.ExpiresAt.Time,
	}

	h.logger.InfoContext(grpcCtx, "Access token refreshed successfully", "email", refreshClaims.Email)
	return WriteJSON(w, http.StatusOK, response)
}

// handleRevokeToken обрабатывает запрос на отзыв сессии пользователя
func (h *authHandler) handleRevokeToken(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling token revocation request")

	// Получаем данные пользователя из контекста, установленного middleware
	claims, ok := r.Context().Value(authContextKey{}).(*token.UserClaims)
	if !ok || claims == nil {
		h.logger.WarnContext(r.Context(), "No auth claims found in context")
		return fmt.Errorf("unauthorized")
	}

	// Создаем gRPC контекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	// Отзываем сессию
	_, err := h.authClient.RevokeSession(grpcCtx, &pbAuth.SessionReq{
		Id: claims.RegisteredClaims.ID,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to revoke session", "session_id", claims.RegisteredClaims.ID, "error", err)
		return fmt.Errorf("error revoking session: %w", err)
	}

	h.logger.InfoContext(grpcCtx, "Session revoked successfully", "email", claims.Email)
	return WriteJSON(w, http.StatusNoContent, nil)
}
