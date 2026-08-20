package server

import (
	"github.com/imabg/authx/internal/validate"
)

func validationErrorMessage(err error) string {
	if err == nil {
		return "invalid request"
	}
	if msg, ok := firstValidationMessage(err); ok {
		return msg
	}
	return err.Error()
}

func firstValidationMessage(err error) (string, bool) {
	return validate.StandardMessages(validate.JSONPath(err), firstValidationTag(err), firstValidationParam(err))
}

func firstValidationTag(err error) string {
	fe, ok := validate.First(err)
	if !ok {
		return ""
	}
	return fe.Tag
}

func firstValidationParam(err error) string {
	fe, ok := validate.First(err)
	if !ok {
		return ""
	}
	return fe.Param
}
