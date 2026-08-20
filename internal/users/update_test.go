package users_test

import (
	"errors"
	"strings"
	"testing"

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

func TestApplyProfileUpdateValidatesLength(t *testing.T) {
	user := users.User{FirstName: "Ada"}
	tooLong := strings.Repeat("a", users.MaxFirstNameLen+1)
	_, err := users.ApplyProfileUpdate(user, users.ProfileUpdate{FirstName: &tooLong})
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
