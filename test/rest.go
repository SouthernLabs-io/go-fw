package test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/middlewares"
	middlewaresmocks "github.com/southernlabs-io/go-fw/middlewares/mocks"
	"github.com/southernlabs-io/go-fw/rest"
)

func NewMockAuthN(t *testing.T, principal middlewares.Principal) fx.Option {
	mockAuthNProvider := middlewaresmocks.NewAuthNProvider(t)
	if principal != nil {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(principal, nil)
	} else {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(nil, middlewares.ErrInvalidToken).Maybe()
	}
	return fx.Supply(fx.Annotate(mockAuthNProvider, fx.As(new(middlewares.AuthNProvider))))
}

var TestModuleRest = rest.Module
