package validate_test

import (
	"strings"
	"testing"

	"github.com/imabg/authx/internal/validate"
)

type profile struct {
	FirstName *string `json:"first_name" validate:"omitempty,runemax=25"`
	LastName  *string `json:"last_name" validate:"omitempty,runemax=50"`
}

type settings struct {
	AuthMethod string `json:"auth_method" validate:"required,oneof=password otp magic_link"`
	Password   struct {
		MinLength int `json:"min_length" validate:"gte=6"`
	} `json:"password"`
}

func TestStructAndJSONPath(t *testing.T) {
	long := strings.Repeat("a", 26)
	err := validate.Struct(profile{FirstName: &long})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if path := validate.JSONPath(err); path != "first_name" {
		t.Fatalf("path = %q", path)
	}
}

func TestSettingsNestedPath(t *testing.T) {
	s := settings{AuthMethod: "password"}
	s.Password.MinLength = 4
	err := validate.Struct(s)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if path := validate.JSONPath(err); path != "password.min_length" {
		t.Fatalf("path = %q", path)
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
			if got := validate.LooksLikeEmail(tt.email); got != tt.want {
				t.Fatalf("LooksLikeEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}
