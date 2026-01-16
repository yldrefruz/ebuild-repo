package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	UserID int64    `json:"user_id"`
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

func NewToken(signingKey []byte, userID int64, scopes []string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(signingKey)
}

func ParseToken(signingKey []byte, tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
	var claims Claims
	_, err := parser.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}
