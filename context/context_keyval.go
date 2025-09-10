package context

import (
	"context"

	"github.com/southernlabs-io/go-fw/sync"
)

var storeKey = CtxKey("_fw_store")

func NewContextWithStore(parent context.Context) context.Context {
	return context.WithValue(parent, storeKey, sync.NewMap[string, any]())
}

func CtxStoreValue(ctx context.Context, key string, value any) bool {
	if store, ok := ctx.Value(storeKey).(*sync.Map[string, any]); ok {
		store.Store(key, value)
		return true
	}
	return false
}

func CtxLoadValue(ctx context.Context, key string) (value any, ok bool) {
	if store, ok := ctx.Value(storeKey).(*sync.Map[string, any]); ok {
		return store.Load(key)
	}
	return nil, false
}

func CtxDeleteKey(ctx context.Context, key string) {
	if store, ok := ctx.Value(storeKey).(*sync.Map[string, any]); ok {
		store.Delete(key)
	}
}
