package app

import (
	"fmt"

	"github.com/imabg/authx/internal/validate"
)

func mapSettingsValidationError(err error) error {
	return validate.Map(err, nil, settingsValidationMessage)
}

func settingsValidationMessage(path, tag, param string) (string, bool) {
	switch path {
	case "auth_method":
		switch tag {
		case "required":
			return "auth_method is required", true
		case "oneof":
			return "auth_method must be password, otp, or magic_link", true
		}
	case "password.min_length":
		if tag == "gte" {
			return "password.min_length must be at least 6", true
		}
	case "otp.length":
		if msg, ok := validate.RangeMessage(path, tag, param, 4, 12); ok {
			return msg, true
		}
	case "otp.ttl_seconds":
		if tag == "gte" && param == "30" {
			return "otp.ttl_seconds must be at least 30", true
		}
		if tag == "lte" && param == "86400" {
			return "otp.ttl_seconds must be at most 86400", true
		}
	case "otp.max_attempts":
		if tag == "gte" {
			return "otp.max_attempts must be at least 1", true
		}
	case "magic_link.ttl_seconds":
		if tag == "gte" {
			return "magic_link.ttl_seconds must be at least 30", true
		}
	case "mail.provider":
		if tag == "oneof" {
			return "mail.provider must be log, sendgrid, or smtp", true
		}
	case "mail.from_email":
		switch tag {
		case "sendgrid_from":
			return "mail.from_email is required for sendgrid", true
		case "smtp_from":
			return "mail.from_email is required for smtp", true
		}
	case "mail.sendgrid.api_key":
		if tag == "required_if" || tag == "secret_set" || tag == "sendgrid_key" {
			return "mail.sendgrid.api_key is required", true
		}
	case "mail.smtp.host":
		if tag == "required_if" || tag == "smtp_host" {
			return "mail.smtp.host is required", true
		}
	case "mail.smtp.port":
		if tag == "gte" || tag == "lte" || tag == "smtp_port" {
			return "mail.smtp.port must be between 1 and 65535", true
		}
	}
	if msg, ok := validate.StandardMessages(path, tag, param); ok {
		return msg, true
	}
	return "", false
}

func validateCreateInput(input CreateInput) error {
	type nameInput struct {
		Name string `validate:"required"`
	}
	if err := validate.Struct(nameInput{Name: input.Name}); err != nil {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateUpdateName(name string) error {
	type nameBody struct {
		Name string `validate:"required"`
	}
	if err := validate.Struct(nameBody{Name: name}); err != nil {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateUpdateStatus(status string) error {
	type statusBody struct {
		Status string `validate:"oneof=active disabled"`
	}
	if err := validate.Struct(statusBody{Status: status}); err != nil {
		return fmt.Errorf("status must be active or disabled")
	}
	return nil
}
