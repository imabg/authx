package token_test

import (
	"strings"
	"testing"
	"time"

	"github.com/imabg/authx/internal/token"
	"github.com/imabg/authx/pkg/config"
)

func testSigner(secret string) *token.JWTSigner {
	cfg := config.ApplicationConfig{}
	cfg.JWT.Secret = secret
	cfg.JWT.Issuer = "authx-test"
	return token.NewJWTSigner(cfg)
}

func TestJWTSigner(t *testing.T) {
	userID := "22222222-2222-2222-2222-222222222222"
	appID := "11111111-1111-1111-1111-111111111111"
	email := "user@example.com"
	signer := testSigner("unit-test-jwt-secret-key-32b")

	tests := []struct {
		name    string
		run     func() error
		wantErr string
	}{
		{
			name: "sign and parse",
			run: func() error {
				raw, err := signer.Sign(userID, appID, email, time.Minute)
				if err != nil {
					return err
				}
				claims, err := signer.Parse(raw)
				if err != nil {
					return err
				}
				if claims.Subject != userID || claims.AppID != appID || claims.Email != email {
					t.Fatalf("claims = %+v", claims)
				}
				return nil
			},
		},
		{
			name: "rejects malformed token",
			run: func() error {
				_, err := signer.Parse("not-a-jwt")
				return err
			},
			wantErr: "token is malformed",
		},
		{
			name: "rejects wrong secret",
			run: func() error {
				raw, err := signer.Sign(userID, appID, email, time.Minute)
				if err != nil {
					return err
				}
				other := testSigner("different-jwt-secret-key-32b")
				_, err = other.Parse(raw)
				return err
			},
			wantErr: "signature is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
