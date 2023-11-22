package token

import (
	"fmt"
	"time"

	"github.com/patrickodacre/accounts-service/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Payload struct {
	Username auth.Username `json:"username"`
	Role     auth.Role     `json:"role"`
	jwt.RegisteredClaims
}

func NewPayload(_username, _role string, _duration time.Duration) (*Payload, error) {
	tokenId, err := uuid.NewRandom()

	if err != nil {
		return nil, err
	}

	username, err := auth.NewUsername(_username)

	if err != nil {
		return nil, err
	}

	role, err := auth.NewRole(_role)

	if err != nil {
		return nil, err
	}

	embeddedClaims := jwt.RegisteredClaims{
		Audience:  []string{"other_service"},
		ID:        tokenId.String(),
		Issuer:    "accounts_service",
		Subject:   "auth_user",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(_duration)),
	}

	payload := &Payload{
		username,
		role,
		embeddedClaims,
	}

	return payload, nil

}

func (p *Payload) Valid() error {
	b, _ := p.ExpiresAt.MarshalJSON()
	fmt.Printf("%s", string(b))
	// if time.Now().After(string(p.ExpiresAt)) {
	// return ErrExpiredToken
	// }
	//
	return nil
}
