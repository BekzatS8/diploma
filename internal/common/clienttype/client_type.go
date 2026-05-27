package clienttype

import (
	"errors"
	"strings"
)

var ErrInvalid = errors.New("invalid client_type")

var allowed = map[string]struct{}{
	"too":            {},
	"ip":             {},
	"representative": {},
}

func Normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func Validate(value string) error {
	if value == "" {
		return nil
	}
	if _, ok := allowed[value]; ok {
		return nil
	}
	return ErrInvalid
}
