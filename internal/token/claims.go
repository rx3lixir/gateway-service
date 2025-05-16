package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserClaims struct {
	Id      int    `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func NewUserClaims(id int, email string, isAdmin bool, duration time.Duration) (*UserClaims, error) {
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("error generating token Id: %w", err)
	}

	return &UserClaims{
		Email:   email,
		Id:      id,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenId.String(),
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		},
	}, nil
}
