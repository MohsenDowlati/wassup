package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type TraceContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

func GenerateTraceContext() (string, error) {
	traceID := make([]byte, 16) // 128-bit
	spanID := make([]byte, 8)   // 64-bit

	if _, err := rand.Read(traceID); err != nil {
		return "", fmt.Errorf("failed to generate trace id: %w", err)
	}
	if _, err := rand.Read(spanID); err != nil {
		return "", fmt.Errorf("failed to generate span id: %w", err)
	}

	return fmt.Sprintf("00-%s-%s-01", hex.EncodeToString(traceID), hex.EncodeToString(spanID)), nil
}

func ParseTraceContext(traceparent string) (*TraceContext, error) {
	if traceparent == "" {
		return nil, nil
	}

	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return nil, fmt.Errorf("malformed traceparent header: expected 4 segments, got %d", len(parts))
	}

	if parts[0] != "00" {
		return nil, fmt.Errorf("unsupported traceparent version: %s", parts[0])
	}

	if len(parts[1]) != 32 || len(parts[2]) != 16 {
		return nil, fmt.Errorf("invalid trace_id or span_id lengths")
	}

	return &TraceContext{
		TraceID: parts[1],
		SpanID:  parts[2],
		Sampled: parts[3] == "01",
	}, nil
}
