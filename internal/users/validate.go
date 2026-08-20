package users

import (
	"fmt"

	"github.com/imabg/authx/internal/validate"
)

func validateProfileUpdate(patch ProfileUpdate) error {
	if err := validate.Struct(patch); err != nil {
		return mapProfileValidationError(err)
	}
	return nil
}

func mapProfileValidationError(err error) error {
	return validate.Map(err, ErrValidation, func(path, tag, param string) (string, bool) {
		switch path {
		case "first_name":
			if tag == "runemax" || tag == "max" {
				return fmt.Sprintf("first_name must be at most %d characters", MaxFirstNameLen), true
			}
		case "last_name":
			if tag == "runemax" || tag == "max" {
				return fmt.Sprintf("last_name must be at most %d characters", MaxLastNameLen), true
			}
		}
		if msg, ok := validate.StandardMessages(path, tag, param); ok {
			return msg, true
		}
		return "", false
	})
}
