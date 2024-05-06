package rest

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/di"
)

type Resource interface {
	Setup(httpHandler core.HTTPHandler)
}

func ProvideAsResource(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[Resource](provider, nil, append(anns, fx.ResultTags(`group:"resources"`)))
}

type Resources []Resource

func NewResources(in struct {
	fx.In
	Resources   []Resource       `group:"resources"`
	HTTPHandler core.HTTPHandler `optional:"true"`
}) Resources {
	Resources(in.Resources).Setup(in.HTTPHandler)
	return in.Resources
}

func (rs Resources) Setup(httpHandler core.HTTPHandler) {
	for _, r := range rs {
		r.Setup(httpHandler)
	}
}

var Module = fx.Invoke(NewResources)
