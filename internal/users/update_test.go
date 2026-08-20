package users_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/imabg/authx/internal/users"
)

func TestApplyProfileUpdate(t *testing.T) {
	user := users.User{FirstName: "Ada", LastName: "Lovelace"}
	first := " Grace "
	last := "Hopper"
	got, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{FirstName: &first, LastName: &last})
	if err != nil {
		t.Fatalf("ApplyProfileUpdate: %v", err)
	}
	if got.FirstName != "Grace" || got.LastName != "Hopper" {
		t.Fatalf("got = %+v", got)
	}
}

func TestApplyProfileUpdateValidatesFirstNameLength(t *testing.T) {
	user := users.User{FirstName: "Ada"}
	tooLong := strings.Repeat("a", users.MaxFirstNameLen+1)
	_, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{FirstName: &tooLong})
	if !errors.Is(err, users.ErrValidation) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "first_name must be at most 25 characters") {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyProfileUpdateValidatesLastNameLength(t *testing.T) {
	user := users.User{LastName: "Lovelace"}
	tooLong := strings.Repeat("b", users.MaxLastNameLen+1)
	_, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{LastName: &tooLong})
	if !errors.Is(err, users.ErrValidation) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "last_name must be at most 50 characters") {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyProfileUpdateValidatesRuneCount(t *testing.T) {
	user := users.User{FirstName: "Ada"}
	// é is one rune but two bytes in UTF-8.
	name := strings.Repeat("é", users.MaxFirstNameLen+1)
	if utf8.RuneCountInString(name) != users.MaxFirstNameLen+1 {
		t.Fatal("test setup: expected rune count over limit")
	}
	_, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{FirstName: &name})
	if !errors.Is(err, users.ErrValidation) {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyProfileUpdatePartial(t *testing.T) {
	user := users.User{FirstName: "Ada", LastName: "Lovelace"}
	last := "Byron"
	got, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{LastName: &last})
	if err != nil {
		t.Fatalf("ApplyProfileUpdate: %v", err)
	}
	if got.FirstName != "Ada" || got.LastName != "Byron" {
		t.Fatalf("got = %+v", got)
	}
}

func TestApplyProfileUpdateAllowsMaxLength(t *testing.T) {
	user := users.User{FirstName: "Ada", LastName: "Lovelace"}
	first := strings.Repeat("a", users.MaxFirstNameLen)
	last := strings.Repeat("b", users.MaxLastNameLen)
	got, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{FirstName: &first, LastName: &last})
	if err != nil {
		t.Fatalf("ApplyProfileUpdate: %v", err)
	}
	if got.FirstName != first || got.LastName != last {
		t.Fatalf("got = %+v", got)
	}
}
