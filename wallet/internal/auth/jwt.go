package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret string
}

func NewTokenManager() *TokenManager {
	s := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	if s == "" {
		s = "dev-secret"
	}
	return &TokenManager{secret: s}
}

func (tm *TokenManager) Generate(wallet string, expiresIn time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"wallet": wallet,
		"exp":    time.Now().Add(expiresIn).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tm.secret))
}

func (tm *TokenManager) Validate(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(tm.secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
