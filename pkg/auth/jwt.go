// Package auth provides JWT issuance/verification and password hashing.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/KhanbalaRashidov/ai-travel-platform/pkg/models"
)

// ErrInvalidToken is returned when a token fails verification.
var ErrInvalidToken = errors.New("invalid token")

// Claims are the JWT custom claims carried in every access token.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTManager issues and verifies HMAC-signed access tokens.
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTManager builds a JWTManager with the given signing secret and TTL.
func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Generate creates a signed token for the given user.
func (m *JWTManager) Generate(u models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		TenantID: u.TenantID,
		Email:    u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Verify parses and validates a token, returning its claims.
func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
