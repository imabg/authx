package validate

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	instance *validator.Validate
)

// V returns the shared validator instance.
func V() *validator.Validate {
	once.Do(func() {
		instance = validator.New()
		instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
			return fld.Name
		})
		_ = instance.RegisterValidation("runemax", validateRuneMax)
		_ = instance.RegisterValidation("authemail", validateAuthEmail)
		_ = instance.RegisterValidation("secret_set", validateSecretSet)
		_ = instance.RegisterValidation("sendgrid_from", validateSendGridFrom)
		_ = instance.RegisterValidation("smtp_from", validateSMTPFrom)
		_ = instance.RegisterValidation("sendgrid_key", validateSendGridKey)
		_ = instance.RegisterValidation("smtp_host", validateSMTPHost)
		_ = instance.RegisterValidation("smtp_port", validateSMTPPort)
	})
	return instance
}

// Struct validates s using the shared validator instance.
func Struct(s any) error {
	if err := V().Struct(s); err != nil {
		return &structError{err: err, root: reflect.TypeOf(s)}
	}
	return nil
}

func validateRuneMax(fl validator.FieldLevel) bool {
	limit, err := strconv.Atoi(fl.Param())
	if err != nil || limit < 0 {
		return false
	}
	return utf8.RuneCountInString(fl.Field().String()) <= limit
}

func validateAuthEmail(fl validator.FieldLevel) bool {
	return LooksLikeEmail(fl.Field().String())
}

func validateSecretSet(fl validator.FieldLevel) bool {
	v := strings.TrimSpace(fl.Field().String())
	return v != "" && v != maskedSecret
}

func mailConfigFromTop(fl validator.FieldLevel) (provider string, ok bool) {
	top := fl.Top()
	if top.Kind() == reflect.Ptr {
		top = top.Elem()
	}
	if top.Kind() != reflect.Struct {
		return "", false
	}
	if field := top.FieldByName("Provider"); field.IsValid() {
		return strings.ToLower(strings.TrimSpace(field.String())), true
	}
	if field := top.FieldByName("Mail"); field.IsValid() {
		mail := field
		if mail.Kind() == reflect.Ptr {
			mail = mail.Elem()
		}
		if mail.Kind() == reflect.Struct {
			if providerField := mail.FieldByName("Provider"); providerField.IsValid() {
				return strings.ToLower(strings.TrimSpace(providerField.String())), true
			}
		}
	}
	return "", false
}

func validateSendGridFrom(fl validator.FieldLevel) bool {
	provider, ok := mailConfigFromTop(fl)
	if !ok || provider != "sendgrid" {
		return true
	}
	return strings.TrimSpace(fl.Field().String()) != ""
}

func validateSMTPFrom(fl validator.FieldLevel) bool {
	provider, ok := mailConfigFromTop(fl)
	if !ok || provider != "smtp" {
		return true
	}
	return strings.TrimSpace(fl.Field().String()) != ""
}

func validateSendGridKey(fl validator.FieldLevel) bool {
	provider, ok := mailConfigFromTop(fl)
	if !ok || provider != "sendgrid" {
		return true
	}
	v := strings.TrimSpace(fl.Field().String())
	return v != "" && v != maskedSecret
}

func validateSMTPHost(fl validator.FieldLevel) bool {
	provider, ok := mailConfigFromTop(fl)
	if !ok || provider != "smtp" {
		return true
	}
	return strings.TrimSpace(fl.Field().String()) != ""
}

func validateSMTPPort(fl validator.FieldLevel) bool {
	provider, ok := mailConfigFromTop(fl)
	if !ok || provider != "smtp" {
		return true
	}
	port := fl.Field().Int()
	return port >= 1 && port <= 65535
}

const maskedSecret = "********"
