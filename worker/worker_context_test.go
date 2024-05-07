package worker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/worker"
)

type ctxKeyType string

const ctxKey ctxKeyType = "ctxKey"

func TestNewWorkerContextWithNameAndID(t *testing.T) {
	initialCtx := context.WithValue(context.Background(), ctxKey, "sentinel")
	ctx := worker.NewWorkerContext(
		initialCtx,
		"test_worker_name",
		"test_worker_id",
	)
	require.NotNil(t, ctx)

	val := ctx.Value(ctxKey).(string)
	require.EqualValues(t, "sentinel", val)
}
