package context

import (
	"context"

	"github.com/southernlabs-io/go-fw/sync"
)

type keyValueContext struct {
	context.Context

	store *sync.Map[string, any]
}

func NewKeyValueContext(parent context.Context) *keyValueContext {
	return &keyValueContext{
		Context: parent,
		store:   sync.NewMap[string, any](),
	}
}

func (c *keyValueContext) Set(key string, value any) {
	c.store.Store(key, value)
}

func (c *keyValueContext) Get(key string) (any, bool) {
	return c.store.Load(key)
}

func (c *keyValueContext) Value(key any) any {
	if keyStr, is := key.(string); is {
		if value, present := c.store.Load(keyStr); present {
			return value
		}
	}
	return c.Context.Value(key)
}

func (c *keyValueContext) Delete(key string) {
	c.store.Delete(key)
}
