package test

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/rest/middleware"
	rest_gin "github.com/southernlabs-io/go-fw/rest_gin"
	middleware_gin "github.com/southernlabs-io/go-fw/rest_gin/middleware"
	middleware_gin_mocks "github.com/southernlabs-io/go-fw/rest_gin/middleware/mocks"

	rest "github.com/southernlabs-io/go-fw/rest"
)

func NewMockAuthNGin(t *testing.T, principal middleware_gin.Principal) fx.Option {
	mockAuthNProvider := middleware_gin_mocks.NewAuthNProvider(t)
	if principal != nil {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(principal, nil).Maybe()
	} else {
		mockAuthNProvider.EXPECT().Authenticate(mock.Anything).Return(nil, middleware.ErrInvalidToken).Maybe()
	}
	return fx.Supply(fx.Annotate(mockAuthNProvider, fx.As(new(middleware.AuthNProvider))))
}

var FxExportRestGin = fx.Options(
	fx.Provide(NewTestHTTPHandlerGin),
	fx.Invoke(rest_gin.NewResources),
)

var FxExportRest = fx.Options(
	middleware.FxExport,
	fx.Provide(rest.NewResources),
	fx.Provide(rest.NewStdServer),
)
