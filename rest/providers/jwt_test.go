package providers

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

func Test(t *testing.T) {
	conf := config.Config{JWT: config.JWTConfig{SigningKey: "dabbad00"}}
	ctx := context.Background()

	jwtSvc := NewJWTProvider(conf)
	claims := jwt.MapClaims{"a": "b", "c": "d"}
	tokenStr, err := jwtSvc.Encode(ctx, claims)
	require.NoError(t, err)
	token, err := jwtSvc.Decode(ctx, tokenStr)
	require.NoError(t, err)
	require.NotNil(t, token)
	require.EqualValues(t, claims, token.Claims)
}

func TestWithExp(t *testing.T) {
	conf := config.Config{JWT: config.JWTConfig{SigningKey: "dabbad00"}}
	ctx := context.Background()

	jwtSvc := NewJWTProvider(conf)
	claims := jwt.MapClaims{"a": "b", "c": "d"}
	tokenStr, err := jwtSvc.EncodeWithExp(ctx, claims, time.Duration(1)*time.Second)
	require.NoError(t, err)
	token, err := jwtSvc.Decode(ctx, tokenStr)
	require.NoError(t, err)
	require.NotNil(t, token)
	mapClaims := token.Claims.(jwt.MapClaims)
	require.False(t, mapClaims.VerifyExpiresAt(time.Now().Add(time.Duration(1)*time.Second).Unix(), true))
}
