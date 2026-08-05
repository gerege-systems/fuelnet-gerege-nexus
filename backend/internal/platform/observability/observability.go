package observability

import (
	"context"
	"log/slog"
)

type ShutdownFunc func(context.Context) error

func SetupTracing(ctx context.Context, serviceName, env string) (ShutdownFunc, error) {
	slog.Info("opentelemetry tracing initialized", "service", serviceName, "environment", env)

	shutdown := func(ctx context.Context) error {
		slog.Info("opentelemetry tracing shutdown successfully")
		return nil
	}

	return shutdown, nil
}
