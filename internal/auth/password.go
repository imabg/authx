package auth

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/imabg/authx/internal/validate"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPayload     = errors.New("invalid_payload_for_auth_method")
	ErrRateLimited        = errors.New("rate limited")
	ErrPasswordPolicy     = errors.New("password does not meet policy")
	ErrValidation         = errors.New("validation error")
	ErrEmailDomainBlocked = errors.New("email domain is not allowed")
)

type PasswordPolicy struct {
	MinLength        int
	RequireMixedCase bool
	RequireDigit     bool
}

func ValidatePassword(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrPasswordPolicy, policy.MinLength)
	}
	if policy.RequireMixedCase {
		var hasUpper, hasLower bool
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
			}
			if unicode.IsLower(r) {
				hasLower = true
			}
		}
		if !hasUpper || !hasLower {
			return fmt.Errorf("%w: must include mixed case", ErrPasswordPolicy)
		}
	}
	if policy.RequireDigit {
		hasDigit := false
		for _, r := range password {
			if unicode.IsDigit(r) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			return fmt.Errorf("%w: must include a digit", ErrPasswordPolicy)
		}
	}
	return nil
}

func LooksLikeEmail(email string) bool {
	return validate.LooksLikeEmail(email)
}
