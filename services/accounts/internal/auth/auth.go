package auth

import (
	"errors"
)

var (
	UsernameTooShort = errors.New("username too short")
	UsernameExists   = errors.New("username exists")
)

const MinUsernameLen = 3

type Username string

func NewUsername(un string) (Username, error) {
	if len(un) < MinUsernameLen {
		return "", UsernameTooShort
	}

	// TODO:: UsernameExists

	return Username(un), nil
}

func (un Username) String() string {
	return string(un)
}

type Role string

func NewRole(r string) (Role, error) {
	// TODO:: Admin-Only
	return Role(r), nil
}

func (r Role) String() string {
	return string(r)
}
