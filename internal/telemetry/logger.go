package telemetry

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const (
	CtxTraceID contextKey = "trace_id"
	CtxUserID  contextKey = "user_id"
)

type ContextualHandler struct {
	slog.Handler
}

func (h *ContextualHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if traceID, ok := ctx.Value(CtxTraceID).(string); ok && traceID != "" {
			r.AddAttrs(slog.String("trace_id", traceID))
		}
		if userID, ok := ctx.Value(CtxUserID).(string); ok && userID != "" {
			r.AddAttrs(slog.String("user_id", userID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func NewProductionLogger(serviceName, env string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	jsonHandler := slog.NewJSONHandler(out, opts).WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
		slog.String("env", env),
	})

	wrappedHandler := &ContextualHandler{Handler: jsonHandler}
	return slog.New(wrappedHandler)
}
