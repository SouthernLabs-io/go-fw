package test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"

	rest "github.com/southernlabs-io/go-fw/rest_gin"
	"github.com/southernlabs-io/go-fw/rest_gin/middleware"
	middlewaremocks "github.com/southernlabs-io/go-fw/rest_gin/middleware/mocks"
)

func NewMockAuthN(t *testing.T, principal middleware.Principal) fx.Option {
	mockAuthNProvider := middlewaremocks.NewAuthNProvider(t)
	if principal != nil {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(principal, nil).Maybe()
	} else {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(nil, middleware.ErrInvalidToken).Maybe()
	}
	return fx.Supply(fx.Annotate(mockAuthNProvider, fx.As(new(middleware.AuthNProvider))))
}

var FxExportRest = fx.Options(
	fx.Provide(NewTestHTTPHandler),
	fx.Invoke(rest.NewResources),
)
