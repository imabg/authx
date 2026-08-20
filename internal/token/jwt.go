package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/imabg/authx/pkg/config"
)

type Claims struct {
	AppID string `json:"app_id"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type JWTSigner struct {
	secret []byte
	issuer string
}

func NewJWTSigner(cfg config.ApplicationConfig) *JWTSigner {
	return &JWTSigner{
		secret: []byte(cfg.JWT.Secret),
		issuer: cfg.JWT.Issuer,
	}
}

func (s *JWTSigner) Sign(userID, appID, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		AppID: appID,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTSigner) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
