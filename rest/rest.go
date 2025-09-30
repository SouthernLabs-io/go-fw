package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/rest/healthcheck"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

func WriteJSON(ctx context.Context, w http.ResponseWriter, statusCode int, obj any) error {
	headers := w.Header()
	headers.Set("Content-Type", "application/json")
	headers.Del("Content-Length") // let net/http set it
	headers.Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		return errors.NewUnknownf("failed to write json response, error: %w", err)
	}
	return nil
}

func WriteJSONStr(ctx context.Context, w http.ResponseWriter, statusCode int, jsonStr string) error {
	headers := w.Header()
	headers.Set("Content-Type", "application/json")
	headers.Del("Content-Length") // let net/http set it
	headers.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	_, err := fmt.Fprintln(w, string(jsonStr))
	if err != nil {
		return errors.NewUnknownf("failed to write json response, error: %w", err)
	}
	return nil
}

var FxExport = fx.Options(
	middleware.FxExport,
	ProvideAsResource(NewHealthCheckFx),
	healthcheck.FxExport,
	fx.Provide(NewResources),
	fx.Provide(NewStdServer),
)
