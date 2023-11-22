package token

import (
	"errors"
)

var (
	ErrTokenInvalid          = errors.New("token is invalid")
	ErrTokenExpired          = errors.New("token is expired")
	ErrTokenUnverifiable     = errors.New("token is unverifiable")
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenSignatureInvalid = errors.New("token signature invalid")
	ErrTokenNotValidYet      = errors.New("token not valid yet")
)
