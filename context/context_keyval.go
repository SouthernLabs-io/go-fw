package context

import (
	"context"

	"github.com/southernlabs-io/go-fw/sync"
)

var storeKey = CtxKey("_fw_store")

type keyValueContext struct {
	context.Context
	store *sync.Map[string, any]
}

func (kvc *keyValueContext) Value(key any) any {
	// If the key is the storeKey, return the keyValueContext itself
	if key == storeKey {
		return kvc
	}
	// If the key is a string, look it up in the store first
	if keyStr, ok := key.(string); ok {
		if value, present := kvc.store.Load(keyStr); present {
			return value
		}
	}
	return kvc.Context.Value(key)
}

func (kvc *keyValueContext) WithValue(key any, value any) context.Context {
	if keyStr, ok := key.(string); ok {
		kvc.store.Store(keyStr, value)
		return kvc
	}
	return context.WithValue(kvc.Context, key, value)
}

func NewContextWithStore(parent context.Context) context.Context {
	return &keyValueContext{
		Context: parent,
		store:   sync.NewMap[string, any](),
	}
}

func CtxStoreValue(ctx context.Context, key string, value any) bool {
	if kvCtx, ok := ctx.Value(storeKey).(*keyValueContext); ok {
		kvCtx.store.Store(key, value)
		return true
	}
	return false
}

func CtxLoadValue(ctx context.Context, key string) (value any, ok bool) {
	if kvCtx, ok := ctx.Value(storeKey).(*keyValueContext); ok {
		return kvCtx.store.Load(key)
	}
	return nil, false
}

func CtxDeleteKey(ctx context.Context, key string) {
	if kvCtx, ok := ctx.Value(storeKey).(*keyValueContext); ok {
		kvCtx.store.Delete(key)
	}
}
