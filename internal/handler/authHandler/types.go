package authhandler

import "time"

// Session представляет собой сессию пользователя
type Session struct {
	ID           string    `json:"id"`
	UserEmail    string    `json:"user_email"`
	RefreshToken string    `json:"refresh_token"`
	IsRevoked    bool      `json:"is_revoked"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// User представляет пользователя для аутентификации
type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"` // Не показываем в JSON
	IsAdmin  bool   `json:"is_admin"`
}

// UserRes представляет информацию о пользователе в ответе
type UserRes struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

// LoginUserReq представляет запрос на вход пользователя
type LoginUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginUserRes представляет ответ на запрос входа пользователя
type LoginUserRes struct {
	SessionID            string    `json:"session_id"`
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	User                 UserRes   `json:"user"`
}

// RenewAccessTokenReq представляет запрос на обновление access token
type RenewAccessTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

// RenewAccessTokenRes представляет ответ на запрос обновления access token
type RenewAccessTokenRes struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}
