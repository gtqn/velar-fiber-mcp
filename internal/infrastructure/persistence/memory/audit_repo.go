// Package memory provides in-memory implementations of persistence ports.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/velar/velar-fiber/internal/domain/entity"
	"go.uber.org/zap"
)

// AuditMemoryRepo implements port.AuditLogger.
// It stores logs in memory for development and also pipes them to Zap logger
// for real-time observability.
type AuditMemoryRepo struct {
	log    *zap.Logger
	mu     sync.RWMutex
	events []entity.AuditLog
}

// NewAuditMemoryRepo creates a new in-memory audit logger.
func NewAuditMemoryRepo(log *zap.Logger) *AuditMemoryRepo {
	return &AuditMemoryRepo{
		log:    log,
		events: make([]entity.AuditLog, 0),
	}
}

// Log records a tool execution event.
func (r *AuditMemoryRepo) Log(ctx context.Context, entry entity.AuditLog) error {
	r.mu.Lock()
	entry.Timestamp = time.Now()
	r.events = append(r.events, entry)
	r.mu.Unlock()

	// High-visibility logging for AI actions
	r.log.Info("AI_ACTION_AUDIT",
		zap.String("trace_id", entry.TraceID),
		zap.String("api_key", entry.APIKeyLabel),
		zap.String("tool", entry.ToolName),
		zap.Any("arguments", entry.Arguments),
		zap.Duration("duration", entry.Duration),
		zap.Bool("success", entry.Success),
	)

	return nil
}

// GetRecent returns the last N audit logs.
func (r *AuditMemoryRepo) GetRecent(ctx context.Context, limit int) ([]entity.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := len(r.events) - limit
	if start < 0 {
		start = 0
	}

	return r.events[start:], nil
}
