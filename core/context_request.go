package core

import (
	"context"
)

var RequestIDCtxKey = ctxKey("lib_request_id")

func GetRequestIDFromCtx(ctx context.Context) string {
	requestID, _ := ctx.Value(RequestIDCtxKey).(string)
	return requestID
}
