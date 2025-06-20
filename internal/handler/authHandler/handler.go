package authhandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
	pbUser "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/user"

	contextkeys "github.com/rx3lixir/gateway-service/pkg/context"
	"github.com/rx3lixir/gateway-service/pkg/logger"
	"github.com/rx3lixir/gateway-service/pkg/password"
	"github.com/rx3lixir/gateway-service/pkg/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authHandler struct {
	authClient pbAuth.AuthServiceClient
	userClient pbUser.UserServiceClient
	tokenMaker *token.JWTMaker
	logger     logger.Logger
}

func NewAuthHandler(authClient pbAuth.AuthServiceClient, userClient pbUser.UserServiceClient, secretKey string, log logger.Logger) *authHandler {
	return &authHandler{
		authClient: authClient,
		userClient: userClient,
		tokenMaker: token.NewJWTMaker(secretKey),
		logger:     log,
	}
}

// setCookieToken устанавливает httpOnly cookie с токеном
func (h *authHandler) setCookieToken(w http.ResponseWriter, name, token string, expires time.Time, httpOnly bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: httpOnly,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// clearCookies очищает все аутентификационные cookies
func (h *authHandler) clearCookies(w http.ResponseWriter) {
	cookies := []string{"access_token", "refresh_token", "session_id"}
	for _, name := range cookies {
		cookie := &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-time.Hour),
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}
}

func (h *authHandler) handleMe(w http.ResponseWriter, r *http.Request) error {
	// Создаем gRPC конетекст
	grpcCtx, cancel := h.createContext(r)
	defer cancel()

	h.logger.InfoContext(grpcCtx, "Recieved request", "handler", "handleMe")

	claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
	if !ok || claims == nil {
		return fmt.Errorf("unauthorized")
	}

	// Получаем пользователя по email из User сервиса
	user, err := h.userClient.GetUser(grpcCtx, &pbUser.UserReq{
		Email: claims.Email,
	})
	if err != nil {
		h.logger.ErrorContext(grpcCtx, "Failed to get user", "email", claims.Email, "error", err)

		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return fmt.Errorf("user not found with email: %s", claims.Email)
		}
		return err
	}

	h.logger.InfoContext(grpcCtx, "Got user", "email", user.Email, "password", user.Password)

	response := struct {
		User                 UserRes   `json:"user"`
		AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	}{
		User: UserRes{
			Name:    user.Name,
			Email:   claims.Email,
			IsAdmin: claims.IsAdmin,
		},
		AccessTokenExpiresAt: claims.RegisteredClaims.ExpiresAt.Time,
	}

	return WriteJSON(w, http.StatusOK, response)
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

	h.logger.InfoContext(grpcCtx, "Got user", "email", user.Email, "password", user.Password)

	// Сравниваем пароль
	ok := password.Verify(loginRequest.Password, user.Password)
	if !ok {
		h.logger.WarnContext(grpcCtx, "Invalid password attempt", "email", loginRequest.Email, "pass", loginRequest.Password)
		return fmt.Errorf("invalid credentials")
	}

	// Проверка может сессия для пользователя уже существует
	h.logger.InfoContext(grpcCtx, "Checking if session for user exists", "email", user.Email, "password", user.Password)

	s, err := h.authClient.GetSessionByEmail(grpcCtx, &pbAuth.GetSessionByEmailReq{
		UserEmail: loginRequest.Email,
	})
	if err != nil {
		st, okStatus := status.FromError(err)
		if okStatus && st.Code() == codes.NotFound {
			// Сессий не найдено. Это НЕ ошибка в данном контексте, а ожидаемое поведение, если пользователь логинится впервые или после долгого перерыва.
			// Продолжаем создавать новую сессию.
			h.logger.InfoContext(grpcCtx, "No existing sessions found for user. Proceeding to create a new session.", "email", loginRequest.Email)
		} else {
			// Произошла другая ошибка при вызове GetSessionByEmail (не codes.NotFound).
			h.logger.ErrorContext(grpcCtx, "Failed to check for existing sessions", "email", loginRequest.Email, "error", err)
			// Это внутренняя ошибка сервера или проблема с сервисом Auth.
			return WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "error checking for existing sessions"})
		}
	} else {
		// Вызов GetSessionByEmail успешен. Сессии могут существовать или список может быть пустым.
		if s != nil && len(s.GetSessions()) > 0 {
			h.logger.InfoContext(grpcCtx, "Found existing sessions for user",
				"email", user.Email, "existing_session_count", len(s.GetSessions()))

			// Если сессий больше 3, отзываем самую старую активную сессию
			if len(s.GetSessions()) > 3 {
				h.logger.InfoContext(grpcCtx, "User has more than 3 sessions, revoking oldest active session")

				// Находим первую неотозванную сессию (предполагаем, что сессии возвращаются в порядке создания)
				for _, sess := range s.GetSessions() {
					if !sess.IsRevoked {
						h.logger.InfoContext(grpcCtx, "Attempting to delete old session", "session_id", sess.Id)
						_, err := h.authClient.DeleteSession(grpcCtx, &pbAuth.SessionReq{
							Id:        sess.Id,
							UserEmail: user.Email,
						})
						if err != nil {
							h.logger.WarnContext(grpcCtx, "Failed to delete existing session",
								"session_id", sess.Id, "error", err)
						} else {
							h.logger.InfoContext(grpcCtx, "Successfully deleted existing session",
								"session_id", sess.Id)
							break // Отозвали одну сессию и выходим
						}
					}
				}
			}
		} else {
			h.logger.InfoContext(grpcCtx, "Successfully checked for sessions. No active pre-existing sessions found or list was empty.",
				"email", user.Email)
		}
	}

	// Создаем токены доступа
	accessToken, accessClaims, err := h.tokenMaker.CreateToken(
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

	// Устанавливаем httpOnly cookies для хранения токенов и id сессии
	h.setCookieToken(w, "access_token", accessToken, refreshClaims.RegisteredClaims.ExpiresAt.Time, true)
	h.setCookieToken(w, "refresh_token", refreshToken, refreshClaims.RegisteredClaims.ExpiresAt.Time, true)
	h.setCookieToken(w, "session_id", session.Id, refreshClaims.RegisteredClaims.ExpiresAt.Time, true)

	// Формируем ответ: возвращаем только публичную информацию о пользователе
	response := LoginUserRes{
		User: UserRes{
			Name:    user.Name,
			Email:   user.Email,
			IsAdmin: user.IsAdmin,
		},
		AccessTokenExpiresAt: accessClaims.RegisteredClaims.ExpiresAt.Time,
	}

	h.logger.InfoContext(grpcCtx, "User logged in successfully", "email", user.Email)
	return WriteJSON(w, http.StatusOK, response)
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling logout request")

	// Читаем session_id из cookie вместо тела запроса
	sessionCookie, err := r.Cookie("session_id")
	if err != nil || sessionCookie.Value == "" {
		// Fallback: пытаемся получить из токена
		claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
		if !ok || claims == nil {
			h.logger.WarnContext(r.Context(), "No session info available for logout")
			return fmt.Errorf("no session information available")
		}

		grpcCtx, cancel := h.createContext(r)
		defer cancel()

		_, err := h.authClient.DeleteSession(grpcCtx, &pbAuth.SessionReq{
			Id: claims.RegisteredClaims.ID,
		})
		if err != nil {
			h.logger.ErrorContext(grpcCtx, "Failed to delete session", "session_id", claims.RegisteredClaims.ID, "error", err)
		}
	} else {
		// Удаляем сессию по ID из cookie
		grpcCtx, cancel := h.createContext(r)
		defer cancel()

		_, err := h.authClient.DeleteSession(grpcCtx, &pbAuth.SessionReq{
			Id: sessionCookie.Value,
		})
		if err != nil {
			h.logger.ErrorContext(grpcCtx, "Failed to delete session", "session_id", sessionCookie.Value, "error", err)
		}
	}

	// Очищаем cookies
	h.clearCookies(w)

	h.logger.InfoContext(r.Context(), "User logged out successfully")
	return WriteJSON(w, http.StatusNoContent, nil)
}

// handleRefreshToken обрабатывает запрос на обновление токена доступа
func (h *authHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "Handling token refresh request")

	// Читаем refresh_token из cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil || refreshCookie.Value == "" {
		h.logger.WarnContext(r.Context(), "No refresh token cookie found")
		return fmt.Errorf("refresh token not found")
	}

	refreshToken := refreshCookie.Value

	// Валидируем refresh token
	refreshClaims, err := h.tokenMaker.VerifyToken(refreshToken)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Invalid refresh token", "error", err, "warning", "invalid cookies will be removed")
		h.clearCookies(w) // Очищаем невалидные cookies
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
		h.clearCookies(w)
		return fmt.Errorf("error getting session: %w", err)
	}

	// Проверяем что сессия не отозвана
	if session.IsRevoked {
		h.logger.WarnContext(grpcCtx, "Session is revoked", "session_id", session.Id)
		h.clearCookies(w)
		return fmt.Errorf("session is revoked")
	}

	// Сверяем email из сессии с email в токене
	if session.UserEmail != refreshClaims.Email {
		h.logger.WarnContext(grpcCtx, "Session email mismatch",
			"token_email", refreshClaims.Email,
			"session_email", session.UserEmail)
		h.clearCookies(w)
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

	// Обновляем cookie с новым access token
	h.setCookieToken(w, "access_token", accessToken, accessClaims.RegisteredClaims.ExpiresAt.Time, true)

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

	// Получаем данные пользователя из контекста
	claims, ok := r.Context().Value(contextkeys.AuthKey).(*token.UserClaims)
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

	// ИСПРАВЛЕНИЕ: Очищаем cookies при отзыве
	h.clearCookies(w)

	h.logger.InfoContext(grpcCtx, "Session revoked successfully", "email", claims.Email)
	return WriteJSON(w, http.StatusNoContent, nil)
}
