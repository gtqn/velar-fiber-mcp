// Package middleware provides Fiber middleware for VELAR-Fiber.
// Each middleware is a pure function with no side effects except logging and
// request augmentation — no global state, no package-level variables.
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/velar/velar-fiber/internal/domain/entity"
	domainerrors "github.com/velar/velar-fiber/internal/domain/errors"
	"github.com/velar/velar-fiber/internal/domain/port"
	"go.uber.org/zap"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey string

const (
	// keyAPIKey is the context key for the authenticated APIKey entity.
	keyAPIKey contextKey = "api_key"
)

// Auth returns a Fiber middleware that validates the X-API-Key header.
// On success it injects the *entity.APIKey into the request context so
// downstream handlers can read the caller's scope without re-querying.
func Auth(repo port.APIKeyRepository, log *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Get("X-API-Key")
		if raw == "" {
			return respondError(c, fiber.StatusUnauthorized,
				domainerrors.New(domainerrors.CodeUnauthorized, "X-API-Key header is required"))
		}

		key, err := repo.FindByValue(c.Context(), raw)
		if err != nil {
			log.Warn("auth failed — invalid API key",
				zap.String("ip", c.IP()),
				zap.String("path", c.Path()),
			)

			return respondError(c, fiber.StatusUnauthorized,
				domainerrors.New(domainerrors.CodeUnauthorized, "invalid API key"))
		}

		// Inject the authenticated key into locals for downstream use.
		c.Locals(string(keyAPIKey), key)

		return c.Next()
	}
}

// GetAPIKey extracts the authenticated APIKey from the Fiber context.
// Returns nil if the Auth middleware was not applied to this route.
func GetAPIKey(c fiber.Ctx) *entity.APIKey {
	val := c.Locals(string(keyAPIKey))
	if val == nil {
		return nil
	}

	key, _ := val.(*entity.APIKey)

	return key
}

// RequireScope returns a middleware that enforces tool-level scope checking.
// Use this inside individual tool handlers when a key has restricted access.
func RequireScope(required entity.ToolScope) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := GetAPIKey(c)
		if key == nil {
			return respondError(c, fiber.StatusUnauthorized,
				domainerrors.New(domainerrors.CodeUnauthorized, "not authenticated"))
		}

		if !key.HasAccessTo(required) {
			return respondError(c, fiber.StatusForbidden,
				domainerrors.New(domainerrors.CodeForbidden,
					"your API key does not have access to the '"+string(required)+"' toolset"))
		}

		return c.Next()
	}
}

// RequestID injects a unique trace ID into every request via X-Request-ID.
// Downstream handlers and the audit log use this ID for correlation.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = generateID()
		}

		c.Set("X-Request-ID", id)
		c.Locals("trace_id", id)

		return c.Next()
	}
}

// Logger returns a middleware that logs every request with latency and status.
func Logger(log *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		log.Info("request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", latency),
			zap.String("ip", c.IP()),
			zap.String("trace_id", c.Locals("trace_id").(string)), //nolint:forcetypeassert
		)

		return err
	}
}

// respondError writes a structured JSON error response.
func respondError(c fiber.Ctx, status int, err *domainerrors.DomainError) error {
	return c.Status(status).JSON(fiber.Map{
		"error": fiber.Map{
			"code":       err.Code,
			"message":    err.Message,
			"field":      err.Field,
			"suggestion": err.Suggestion,
		},
	})
}

// generateID creates a lightweight correlation ID without importing crypto/rand.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}
