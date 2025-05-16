package authhandler

import (
	"context"
	"log/slog"
	"net/http"

	pbAuth "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/auth"
)

type authHandler struct {
	client pbAuth.AuthServiceClient
	logger *slog.Logger
}

func NewEventHandler(client pbAuth.AuthServiceClient, ctx context.Context, log *slog.Logger) *authHandler {
	return &authHandler{
		client: client,
		logger: log,
	}
}

func (h *authHandler) handleLoginUser(w http.ResponseWriter, r *http.Request) error {
	h.logger.InfoContext(r.Context(), "handling login user")

	loginRequest := new(models.LoginUserReq)

	if err := json.NewDecoder(r.Body).Decode(loginRequest); err != nil {
		WriteJSON(w, http.StatusBadRequest, "error decoding request body")
		return err
	}

	user, err := s.store.GetUserByEmail(s.dbContext, loginRequest.Email)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, "no user found with such email")
		return err
	}

	ok := password.Verify(loginRequest.Password, user.Password)
	if !ok {
		WriteJSON(w, http.StatusUnauthorized, "password is invalid")
		return err
	}

	accessToken, accessClaims, err := s.TokenMaker.CreateToken(user.Id, user.Email, user.IsAdmin, time.Minute*15)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error creating token")
		return err
	}

	refreshToken, refreshClaims, err := s.TokenMaker.CreateToken(user.Id, user.Email, user.IsAdmin, time.Hour*24)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error creating token")
		return err
	}

	session, err := s.sessions.CreateSession(s.dbContext, &models.Session{
		Id:           refreshClaims.RegisteredClaims.ID,
		UserEmail:    user.Email,
		RefreshToken: refreshToken,
		IsRevoked:    false,
		ExpiresAt:    refreshClaims.RegisteredClaims.ExpiresAt.Time,
	})
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error creating session")
		return err
	}

	res := models.LoginUserRes{
		SessionId:            session.Id,
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: accessClaims.RegisteredClaims.ExpiresAt.Time,
		User: models.GetUserRes{
			Name:    user.Name,
			Email:   user.Email,
			IsAdmin: user.IsAdmin,
		},
	}

	WriteJSON(w, http.StatusOK, res)

	return nil
}

func (s *APIServer) handleLogoutUser(w http.ResponseWriter, r *http.Request) error {
	claims := r.Context().Value(authKey{}).(*token.UserClaims)

	err := s.sessions.DeleteSession(s.dbContext, claims.RegisteredClaims.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error deleting session")
		return err
	}

	WriteJSON(w, http.StatusNoContent, "deleted successfully")

	return nil
}

func (s *APIServer) handleRenewAcessToken(w http.ResponseWriter, r *http.Request) error {
	req := new(models.RenewAccessTokenReq)

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		WriteJSON(w, http.StatusBadRequest, "error decoding request body")
		return err
	}

	// Проверка, не находится ли токен в черном списке
	isBlacklisted, err := s.sessions.IsTokenBlacklisted(s.dbContext, req.RefershToken)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error checking token status")
		return err
	}

	if isBlacklisted {
		WriteJSON(w, http.StatusUnauthorized, "token is revoked")
		return fmt.Errorf("token is blacklisted")
	}

	refreshClaims, err := s.TokenMaker.VerifyToken(req.RefershToken)
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, "error verifying token")
		return err
	}

	session, err := s.sessions.GetSession(s.dbContext, refreshClaims.RegisteredClaims.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error getting session")
		return err
	}

	if session.IsRevoked {
		WriteJSON(w, http.StatusUnauthorized, "session is already revoked")
		return nil
	}

	if session.UserEmail != refreshClaims.Email {
		WriteJSON(w, http.StatusUnauthorized, "session is invalid")
		return nil
	}

	accessToken, accessClaims, err := s.TokenMaker.CreateToken(refreshClaims.Id, refreshClaims.Email, refreshClaims.IsAdmin, time.Minute*15)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error creating token")
		return nil
	}

	res := models.RenewAccessTokenRes{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessClaims.RegisteredClaims.ExpiresAt.Time,
	}

	WriteJSON(w, http.StatusOK, res)

	return nil
}

func (s *APIServer) handleRevokeSession(w http.ResponseWriter, r *http.Request) error {
	claims := r.Context().Value(authKey{}).(*token.UserClaims)

	err := s.sessions.RevokeSession(s.dbContext, claims.RegisteredClaims.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, "error revoking session")
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
