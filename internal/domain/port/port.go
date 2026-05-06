// Package port defines the interfaces (ports) of the hexagonal architecture.
// Adapters in the infrastructure layer implement these interfaces.
// The domain and application layers depend only on these interfaces — never
// on concrete implementations.
package port

import (
	"context"

	"github.com/velar/velar-fiber/internal/domain/entity"
)

// APIKeyRepository is the read port for looking up API key metadata.
// Implementations: infrastructure/persistence/memory.APIKeyMemoryRepo
type APIKeyRepository interface {
	// FindByValue returns the APIKey for the given raw key string.
	// Returns domain error CodeUnauthorized if the key does not exist.
	FindByValue(ctx context.Context, value string) (*entity.APIKey, error)
}

// AuditLogger is the write port for emitting audit events.
// Every tool call should be logged for observability and security.
type AuditLogger interface {
	// Log records a tool execution event.
	Log(ctx context.Context, entry entity.AuditLog) error
	// GetRecent returns the last N audit logs.
	GetRecent(ctx context.Context, limit int) ([]entity.AuditLog, error)
}
