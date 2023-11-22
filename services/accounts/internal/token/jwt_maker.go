package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeySize = 32

var (
	ErrInvalidKeyLength = errors.New(fmt.Sprintf("invalid key size: must be at least %d characters", minSecretKeySize))
)

// impls Maker interface
type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {

	if len(secretKey) < minSecretKeySize {
		return nil, ErrInvalidKeyLength
	}

	return &JWTMaker{secretKey}, nil
}

func (m *JWTMaker) CreateToken(username, role string, duration time.Duration) (string, *Payload, error) {
	// payload == claims
	payload, err := NewPayload(username, role, duration)

	if err != nil {
		return "", payload, err
	}

	_token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	token, err := _token.SignedString([]byte(m.secretKey))

	return token, payload, nil
}

func (m *JWTMaker) VerifyToken(_token string) (*Payload, error) {
	keyfunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)

		if !ok {
			return nil, ErrTokenInvalid
		}

		return []byte(m.secretKey), nil
	}

	token, err := jwt.ParseWithClaims(_token, &Payload{}, keyfunc)

	switch {
	case token.Valid:
		payload, ok := token.Claims.(*Payload)

		if !ok {
			return nil, ErrTokenInvalid
		}

		return payload, nil

	// TODO:: Create custom Errors for each
	case errors.Is(err, jwt.ErrTokenMalformed):
		return nil, ErrTokenMalformed
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return nil, ErrTokenSignatureInvalid
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return nil, ErrTokenNotValidYet
	default:
		return nil, ErrTokenExpired
	}
}
