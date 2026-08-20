package hasher_test

import (
	"testing"

	"github.com/imabg/authx/internal/hasher"
)

func TestHashAndCompare(t *testing.T) {
	hash, err := hasher.Hash("ValidPass1")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	tests := []struct {
		name  string
		plain string
		want  bool
	}{
		{name: "matching password", plain: "ValidPass1", want: true},
		{name: "wrong password", plain: "WrongPass1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hasher.Compare(tt.plain, hash)
			if err != nil {
				t.Fatalf("Compare: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Compare = %v, want %v", got, tt.want)
			}
		})
	}
}
