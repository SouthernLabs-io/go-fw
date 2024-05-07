package providers

import (
	"context"
	"encoding/hex"
	"reflect"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

var signingMethod = jwt.SigningMethodHS256

type JWTProvider struct {
	key    []byte
	parser *jwt.Parser
}

func NewJWTProvider(conf core.Config) JWTProvider {
	if conf.JWT.SigningKey == "" {
		panic(errors.Newf(errors.ErrCodeBadState, "JWT signing key not set in config"))
	}

	key, err := hex.DecodeString(conf.JWT.SigningKey)
	if err != nil {
		panic(errors.NewUnknownf("could not hex decode signing key, error: %w", err))
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{signingMethod.Name}))

	return JWTProvider{key, parser}
}

func (s JWTProvider) EncodeWithExp(ctx context.Context, claims jwt.Claims, ttl time.Duration) (string, error) {
	exp := jwt.NewNumericDate(time.Now().Add(ttl))
	switch v := claims.(type) {
	case jwt.MapClaims:
		v["exp"] = exp
	case *jwt.RegisteredClaims:
		v.ExpiresAt = exp
	case interface {
		SetExpiresAt(*jwt.NumericDate)
	}:
		v.SetExpiresAt(exp)
	default:
		return "", errors.Newf(errors.ErrCodeBadArgument, "could not set expires on type: "+reflect.TypeOf(claims).String())
	}

	return s.Encode(ctx, claims)
}

func (s JWTProvider) Encode(_ context.Context, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(signingMethod, claims)
	tokenStr, err := token.SignedString(s.key)
	if err != nil {
		return "", errors.NewUnknownf("failed to encode, error: %w", err)
	}
	return tokenStr, nil
}

func (s JWTProvider) Decode(_ context.Context, tokenStr string) (*jwt.Token, error) {
	token, err := s.parser.Parse(tokenStr, s.getSigningKey)
	if err != nil {
		return nil, errors.NewUnknownf("failed to decode token: %s, error: %w", tokenStr, err)
	}
	return token, nil
}

func (s JWTProvider) DecodeWithRegisteredClaims(ctx context.Context, tokenStr string) (*jwt.Token, error) {
	return s.DecodeWithCustomRegisteredClaims(ctx, tokenStr, &jwt.RegisteredClaims{})
}

func (s JWTProvider) DecodeWithCustomRegisteredClaims(_ context.Context, tokenStr string, registeredClaims jwt.Claims) (*jwt.Token, error) {
	token, err := s.parser.ParseWithClaims(tokenStr, registeredClaims, s.getSigningKey)
	if err != nil {
		return nil, errors.NewUnknownf("failed to decode token: %s, error: %w", tokenStr, err)
	}
	return token, nil
}

func (s JWTProvider) getSigningKey(_ *jwt.Token) (any, error) {
	return s.key, nil
}

var JWTProviderModule = fx.Provide(NewJWTProvider)
