package rest

import (
	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
)

type Resource interface {
	Setup(httpHandler lib.HTTPHandler)
}

func ProvideAsResource(provider any, anns ...fx.Annotation) fx.Option {
	return lib.FxProvideAs[Resource](provider, nil, append(anns, fx.ResultTags(`group:"resources"`)))
}

type Resources []Resource

func NewResources(in struct {
	fx.In
	Resources   []Resource      `group:"resources"`
	HTTPHandler lib.HTTPHandler `optional:"true"`
}) Resources {
	Resources(in.Resources).Setup(in.HTTPHandler)
	return in.Resources
}

func (rs Resources) Setup(httpHandler lib.HTTPHandler) {
	for _, r := range rs {
		r.Setup(httpHandler)
	}
}

var Module = fx.Invoke(NewResources)
