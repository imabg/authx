package users

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxFirstNameLen = 25
	MaxLastNameLen  = 50
)

var ErrValidation = errors.New("user validation")

type ProfileUpdate struct {
	FirstName *string
	LastName  *string
}

func ApplyProfileUpdate(user User, patch ProfileUpdate) (User, error) {
	if patch.FirstName != nil {
		v := strings.TrimSpace(*patch.FirstName)
		if utf8.RuneCountInString(v) > MaxFirstNameLen {
			return User{}, fmt.Errorf("%w: first_name must be at most %d characters", ErrValidation, MaxFirstNameLen)
		}
		user.FirstName = v
	}
	if patch.LastName != nil {
		v := strings.TrimSpace(*patch.LastName)
		if utf8.RuneCountInString(v) > MaxLastNameLen {
			return User{}, fmt.Errorf("%w: last_name must be at most %d characters", ErrValidation, MaxLastNameLen)
		}
		user.LastName = v
	}
	return user, nil
}
