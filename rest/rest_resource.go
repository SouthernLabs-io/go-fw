package rest

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/di"
)

type Resource interface {
	Register(srv StdServer)
}

func ProvideAsResource(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[Resource](provider, nil, append(anns, fx.ResultTags(`group:"rest_resources"`)))
}

type Resources []Resource

func NewResources(in struct {
	fx.In
	Resources []Resource `group:"rest_resources"`
}) Resources {
	return in.Resources
}
