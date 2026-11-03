package validation

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// regexTikTokUsername accepts an optional leading @ and 2-24 letters, digits,
// underscores or periods.
var regexTikTokUsername = regexp.MustCompile(`^@?[A-Za-z0-9._]{2,24}$`)

func validateTikTokUsername(fl validator.FieldLevel) bool {
	return regexTikTokUsername.MatchString(strings.TrimSpace(fl.Field().String()))
}

func validateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}
