// Package logger provides a structured, high-performance logger built on
// Uber Zap. It is the ONLY logging mechanism in VELAR-Fiber — fmt.Print
// and log.Print are forbidden by the linter configuration.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a production-ready Zap logger.
// In development mode it outputs human-readable colored console logs.
// In production mode it outputs structured JSON lines.
func New(env string) (*zap.Logger, error) {
	var cfg zap.Config

	if env == "development" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		// Always output JSON in production — parseable by log aggregators.
		cfg.Encoding = "json"
		cfg.EncoderConfig.TimeKey = "ts"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	log, err := cfg.Build(
		zap.AddCallerSkip(0),
		// Add service name to every log line.
		zap.Fields(zap.String("service", "velar-fiber")),
	)
	if err != nil {
		return nil, fmt.Errorf("building zap logger: %w", err)
	}

	return log, nil
}

// Sync flushes any buffered log entries. Call this at application shutdown.
// It is safe to call even if the logger is nil.
func Sync(log *zap.Logger) {
	if log == nil {
		return
	}
	// Error is intentionally ignored — flushing on exit is best-effort.
	_ = log.Sync()
}
