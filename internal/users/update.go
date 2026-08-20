package users

import (
	"errors"
	"strings"
)

const (
	MaxFirstNameLen = 25
	MaxLastNameLen  = 50
)

var ErrValidation = errors.New("user validation")

type ProfileUpdate struct {
	FirstName *string `json:"first_name" validate:"omitempty,runemax=25"`
	LastName  *string `json:"last_name" validate:"omitempty,runemax=50"`
}

func ApplyProfileUpdate(user User, patch ProfileUpdate) (User, error) {
	validated := ProfileUpdate{}
	if patch.FirstName != nil {
		v := strings.TrimSpace(*patch.FirstName)
		validated.FirstName = &v
	}
	if patch.LastName != nil {
		v := strings.TrimSpace(*patch.LastName)
		validated.LastName = &v
	}
	if err := validateProfileUpdate(validated); err != nil {
		return User{}, err
	}
	if validated.FirstName != nil {
		user.FirstName = *validated.FirstName
	}
	if validated.LastName != nil {
		user.LastName = *validated.LastName
	}
	return user, nil
}
