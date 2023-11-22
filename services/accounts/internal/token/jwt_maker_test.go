package token

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickodacre/accounts-service/internal/token"
	"github.com/patrickodacre/accounts-service/internal/util"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {

	maker, err := token.NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomName()
	role := util.User
	duration := time.Minute

	// issuedAt := time.Now()
	// expiredAt := time.Now().Add(duration)

	token, payload, err := maker.CreateToken(username, role, duration)

	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)

	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username.String())
	require.Equal(t, role, payload.Role.String())
	b, err := payload.IssuedAt.MarshalJSON()
	fmt.Printf("TIME :: %s", string(b))
	// require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	// require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := token.NewJWTMaker(util.RandomString(32))
	username := util.RandomName()
	role := util.User
	duration := time.Minute

	// issuedAt := time.Now()
	// expiredAt := time.Now().Sub(duration)

	// negative duration to trick it into being expired
	_token, payload, err := maker.CreateToken(username, role, -duration)

	require.NoError(t, err)
	require.NotEmpty(t, _token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(_token)

	require.Error(t, err)
	require.EqualError(t, err, jwt.ErrTokenExpired.Error())
	require.Nil(t, payload)
}

func TestInvalidToken(t *testing.T) {
	username := util.RandomName()
	role := util.User
	duration := time.Minute

	_payload, err := token.NewPayload(username, role, duration)
	require.NoError(t, err)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, _payload)
	_token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := token.NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	payload, err := maker.VerifyToken(_token)
	require.EqualError(t, err, jwt.ErrTokenExpired.Error())
	require.Nil(t, payload)

}
