package auth

import (
	"errors"
	"testing"
	"time"
)

func TestValidatePassword(t *testing.T) {
	policy := PasswordPolicy{MinLength: 8, RequireMixedCase: true, RequireDigit: true}
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "too short", password: "Short1", wantErr: true},
		{name: "missing mixed case", password: "alllowercase1", wantErr: true},
		{name: "missing digit", password: "NoDigitsHere", wantErr: true},
		{name: "valid", password: "ValidPass1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if tt.wantErr {
				if !errors.Is(err, ErrPasswordPolicy) {
					t.Fatalf("error = %v, want %v", err, ErrPasswordPolicy)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLooksLikeEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{email: "user@example.com", want: true},
		{email: "not-an-email", want: false},
		{email: "@example.com", want: false},
		{email: "user@", want: false},
		{email: "Ada <user@example.com>", want: false},
		{email: "user@localhost", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := LooksLikeEmail(tt.email); got != tt.want {
				t.Fatalf("LooksLikeEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	if !limiter.Allow("k") || !limiter.Allow("k") {
		t.Fatal("first two should be allowed")
	}
	if limiter.Allow("k") {
		t.Fatal("third should be blocked")
	}
}
