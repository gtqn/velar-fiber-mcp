// Package memory provides in-memory implementations of the domain ports.
// These implementations are used in development and testing.
// Swapping to a database (Postgres, Redis, etc.) only requires adding a
// new adapter in the infrastructure layer — zero domain code changes.
package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/velar/velar-fiber/internal/domain/entity"
	domainerrors "github.com/velar/velar-fiber/internal/domain/errors"
)

// APIKeyMemoryRepo implements port.APIKeyRepository using a simple map.
// It is populated at startup from the API_KEYS environment variable.
type APIKeyMemoryRepo struct {
	keys map[string]*entity.APIKey
}

// NewAPIKeyMemoryRepo parses the raw key string and builds the in-memory store.
// Format: "keyValue:scope,keyValue2:scope2"
// Example: "sk-prod-1:all,sk-readonly-1:docs,utils"
func NewAPIKeyMemoryRepo(raw string) (*APIKeyMemoryRepo, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("API_KEYS is empty — server has no valid keys")
	}

	repo := &APIKeyMemoryRepo{keys: make(map[string]*entity.APIKey)}

	pairs := strings.Split(raw, ",")
	for i, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, ":", 2)

		if len(parts) != 2 {
			return nil, fmt.Errorf("API_KEYS entry #%d %q must be in format key:scope", i+1, pair)
		}

		keyVal := strings.TrimSpace(parts[0])
		scope := entity.ToolScope(strings.TrimSpace(parts[1]))

		if !isValidScope(scope) {
			return nil, fmt.Errorf("unknown scope %q for key #%d — valid: all,github,docs,system,web,utils", scope, i+1)
		}

		repo.keys[keyVal] = &entity.APIKey{
			Value: keyVal,
			Scope: scope,
			Label: fmt.Sprintf("key-%d", i+1),
		}
	}

	return repo, nil
}

// FindByValue looks up an API key by its raw value.
func (r *APIKeyMemoryRepo) FindByValue(_ context.Context, value string) (*entity.APIKey, error) {
	key, ok := r.keys[value]
	if !ok {
		return nil, domainerrors.New(domainerrors.CodeUnauthorized, "invalid or unknown API key")
	}

	return key, nil
}

// isValidScope returns true when scope is one of the defined ToolScope values.
func isValidScope(s entity.ToolScope) bool {
	switch s {
	case entity.ScopeAll, entity.ScopeGitHub, entity.ScopeDocs,
		entity.ScopeSystem, entity.ScopeWeb, entity.ScopeUtils, entity.ScopeFiber:
		return true
	default:
		return false
	}
}
