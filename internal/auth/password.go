package auth

import (
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// PasswordCost is bcrypt's work factor. About a quarter of a second per hash
// on a small VPS: slow for guessing, still inside the sign-in budget.
const PasswordCost = 12

// MinPassword is NIST's floor. bcrypt reads at most 72 bytes, so longer
// passwords are refused rather than silently cut.
const (
	MinPassword = 8
	MaxPassword = 72
)

// ErrBadEmail and ErrBadPassword are user input errors.
var (
	ErrBadEmail    = errors.New("enter a valid email address")
	ErrBadPassword = errors.New("password must be 8 to 72 characters")
)

// dummyHash is compared when an email has no account, so a miss costs the
// same time as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("sitewise-no-account"), PasswordCost)

// NormalEmail trims and lower-cases an address and checks its shape.
func NormalEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 || strings.Count(email, "@") != 1 {
		return "", ErrBadEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.Index(email, "@"):], ".") {
		return "", ErrBadEmail
	}
	return email, nil
}

// CheckPasswordRule refuses passwords bcrypt cannot hold whole.
func CheckPasswordRule(password string) error {
	if len(password) < MinPassword || len(password) > MaxPassword {
		return ErrBadPassword
	}
	return nil
}

// HashPassword returns the bcrypt hash stored on the user.
func HashPassword(password string) ([]byte, error) {
	if err := CheckPasswordRule(password); err != nil {
		return nil, err
	}
	return bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
}

// PasswordMatches compares in constant time. A nil hash means no account and
// still spends one compare.
func PasswordMatches(hash []byte, password string) bool {
	if len(hash) == 0 {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}
