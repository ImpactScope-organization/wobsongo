package auth

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// defaultCost is the bcrypt cost factor used in production.
const defaultCost = 12

type hashOptions struct {
	cost int
}

type hashOption func(*hashOptions)

// WithMinCost sets bcrypt cost to the minimum (4).
// Use in tests and seed scripts to avoid slow hashing at high iteration counts.
func WithMinCost() hashOption {
	return func(opts *hashOptions) {
		opts.cost = bcrypt.MinCost
	}
}

const (
	// PasswordMinLength is the fewest characters a password may have.
	PasswordMinLength = 12
	// PasswordMaxBytes is bcrypt's input limit; longer passwords are rejected
	// rather than silently truncated.
	PasswordMaxBytes = 72
)

// PasswordValid reports whether password meets the policy: at least
// PasswordMinLength characters and at most PasswordMaxBytes bytes.
func PasswordValid(password string) bool {
	return utf8.RuneCountInString(password) >= PasswordMinLength &&
		len(password) <= PasswordMaxBytes
}

// NormalizeEmail trims and lowercases an email address, so the same address
// can't be registered twice in different cases.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string, options ...hashOption) (string, error) {
	opts := &hashOptions{cost: defaultCost}
	for _, o := range options {
		o(opts)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), opts.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePasswords returns nil if the plaintext password matches the hash.
func ComparePasswords(password, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}
