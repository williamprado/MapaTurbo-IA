package validator

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func InitValidator() {
	Validate = validator.New()
}

func FormatValidationError(err error) map[string]string {
	errs := make(map[string]string)
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			errs[fe.Field()] = msgForTag(fe.Tag(), fe.Param())
		}
	} else {
		errs["error"] = err.Error()
	}
	return errs
}

func msgForTag(tag string, param string) string {
	switch tag {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address"
	case "min":
		return fmt.Sprintf("Must be at least %s characters long", param)
	case "max":
		return fmt.Sprintf("Must be at most %s characters long", param)
	default:
		return "Invalid value"
	}
}
