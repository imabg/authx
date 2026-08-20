package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type structError struct {
	err  error
	root reflect.Type
}

func (e *structError) Error() string {
	return e.err.Error()
}

func (e *structError) Unwrap() error {
	return e.err
}

// FieldError describes the first validation failure.
type FieldError struct {
	Field string
	Tag   string
	Param string
}

func validationErrors(err error) (validator.ValidationErrors, reflect.Type, bool) {
	var wrapped *structError
	if errors.As(err, &wrapped) {
		var verrs validator.ValidationErrors
		if errors.As(wrapped.err, &verrs) {
			return verrs, wrapped.root, true
		}
		return nil, wrapped.root, false
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return verrs, nil, true
	}
	return nil, nil, false
}

// First returns the first validation error detail when err is a validator error.
func First(err error) (FieldError, bool) {
	verrs, _, ok := validationErrors(err)
	if !ok || len(verrs) == 0 {
		return FieldError{}, false
	}
	fe := verrs[0]
	return FieldError{
		Field: fe.Field(),
		Tag:   fe.Tag(),
		Param: fe.Param(),
	}, true
}

// JSONPath returns a dotted json field path such as password.min_length.
func JSONPath(err error) string {
	verrs, root, ok := validationErrors(err)
	if !ok || len(verrs) == 0 {
		return ""
	}
	return structNamespaceToJSON(verrs[0], root)
}

func structNamespaceToJSON(fe validator.FieldError, root reflect.Type) string {
	parts := strings.Split(fe.StructNamespace(), ".")
	if len(parts) <= 1 {
		return fe.Field()
	}
	if root == nil {
		return fe.Field()
	}
	current := root
	if current.Kind() == reflect.Ptr {
		current = current.Elem()
	}
	out := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		if current.Kind() != reflect.Struct {
			break
		}
		sf, ok := current.FieldByName(parts[i])
		if !ok {
			return fe.Field()
		}
		name := strings.SplitN(sf.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			name = parts[i]
		}
		out = append(out, name)
		current = sf.Type
		for current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
	}
	if len(out) == 0 {
		return fe.Field()
	}
	return strings.Join(out, ".")
}

// MessageFunc maps a validation failure to a client-facing message.
type MessageFunc func(path, tag, param string) (string, bool)

// Map wraps err with sentinel when fn returns false for the mapped message.
func Map(err error, sentinel error, fn MessageFunc) error {
	if err == nil {
		return nil
	}
	path := JSONPath(err)
	fe, ok := First(err)
	if !ok {
		if sentinel != nil {
			return fmt.Errorf("%w: %v", sentinel, err)
		}
		return err
	}
	if fn != nil {
		if msg, ok := fn(path, fe.Tag, fe.Param); ok {
			if sentinel != nil {
				return fmt.Errorf("%w: %s", sentinel, msg)
			}
			return fmt.Errorf("%s", msg)
		}
	}
	if sentinel != nil {
		return fmt.Errorf("%w: %s is invalid", sentinel, path)
	}
	return fmt.Errorf("%s is invalid", path)
}

// StandardMessages maps common validator tags to dotted-path messages.
func StandardMessages(path, tag, param string) (string, bool) {
	switch tag {
	case "required":
		return path + " is required", true
	case "required_if":
		return path + " is required", true
	case "oneof":
		return path + " must be one of: " + strings.ReplaceAll(param, " ", ", "), true
	case "email", "authemail":
		if path == "email" {
			return "email is invalid", true
		}
		return path + " is invalid", true
	case "gte":
		return path + " must be at least " + param, true
	case "lte":
		return path + " must be at most " + param, true
	case "min", "max", "runemax":
		if tag == "runemax" || tag == "max" {
			return path + " must be at most " + param + " characters", true
		}
		return path + " must be at least " + param + " characters", true
	case "secret_set":
		return path + " is required", true
	}
	return "", false
}

// RangeMessage returns "path must be between min and max" for paired gte/lte failures.
func RangeMessage(path, tag, param string, min, max int) (string, bool) {
	if path == "" {
		return "", false
	}
	switch tag {
	case "gte":
		if param == fmt.Sprintf("%d", min) {
			return fmt.Sprintf("%s must be between %d and %d", path, min, max), true
		}
	case "lte":
		if param == fmt.Sprintf("%d", max) {
			return fmt.Sprintf("%s must be between %d and %d", path, min, max), true
		}
	}
	return "", false
}
