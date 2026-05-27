package auth

import "buhpro/internal/common/clienttype"

var ErrInvalidClientType = clienttype.ErrInvalid

func NormalizeClientType(value string) string {
	return clienttype.Normalize(value)
}

func ValidateClientType(value string) error {
	return clienttype.Validate(value)
}
