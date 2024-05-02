// Code generated by mockery v2.42.1. DO NOT EDIT.

package mocks

import (
	gin "github.com/gin-gonic/gin"
	middlewares "github.com/southernlabs-io/go-fw/middlewares"
	mock "github.com/stretchr/testify/mock"
)

// AuthNProvider is an autogenerated mock type for the AuthNProvider type
type AuthNProvider struct {
	mock.Mock
}

type AuthNProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *AuthNProvider) EXPECT() *AuthNProvider_Expecter {
	return &AuthNProvider_Expecter{mock: &_m.Mock}
}

// Authenticate provides a mock function with given fields: ctx
func (_m *AuthNProvider) Authenticate(ctx *gin.Context) (middlewares.Principal, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Authenticate")
	}

	var r0 middlewares.Principal
	var r1 error
	if rf, ok := ret.Get(0).(func(*gin.Context) (middlewares.Principal, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(*gin.Context) middlewares.Principal); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middlewares.Principal)
		}
	}

	if rf, ok := ret.Get(1).(func(*gin.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AuthNProvider_Authenticate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Authenticate'
type AuthNProvider_Authenticate_Call struct {
	*mock.Call
}

// Authenticate is a helper method to define mock.On call
//   - ctx *gin.Context
func (_e *AuthNProvider_Expecter) Authenticate(ctx interface{}) *AuthNProvider_Authenticate_Call {
	return &AuthNProvider_Authenticate_Call{Call: _e.mock.On("Authenticate", ctx)}
}

func (_c *AuthNProvider_Authenticate_Call) Run(run func(ctx *gin.Context)) *AuthNProvider_Authenticate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*gin.Context))
	})
	return _c
}

func (_c *AuthNProvider_Authenticate_Call) Return(_a0 middlewares.Principal, _a1 error) *AuthNProvider_Authenticate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *AuthNProvider_Authenticate_Call) RunAndReturn(run func(*gin.Context) (middlewares.Principal, error)) *AuthNProvider_Authenticate_Call {
	_c.Call.Return(run)
	return _c
}

// NewAuthNProvider creates a new instance of AuthNProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAuthNProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *AuthNProvider {
	mock := &AuthNProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}