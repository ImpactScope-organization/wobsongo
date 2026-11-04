package validation

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

func validateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}
