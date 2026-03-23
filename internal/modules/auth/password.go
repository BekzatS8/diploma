package auth

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

var (
	upperRe = regexp.MustCompile(`[A-Z]`)
	lowerRe = regexp.MustCompile(`[a-z]`)
	digitRe = regexp.MustCompile(`[0-9]`)
)

func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return errors.New("password must be at least 8 characters")
	}
	if !upperRe.MatchString(password) || !lowerRe.MatchString(password) || !digitRe.MatchString(password) {
		return errors.New("password must include upper, lower, and numeric characters")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
