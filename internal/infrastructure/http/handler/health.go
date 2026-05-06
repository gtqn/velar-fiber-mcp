// Package handler provides the HTTP handlers that bridge Fiber and the MCP server.
package handler

import (
	"github.com/gofiber/fiber/v3"
)

// HealthHandler handles readiness and liveness probes.
// These endpoints are intentionally unauthenticated — load balancers
// and container orchestrators call them without credentials.
type HealthHandler struct{}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Liveness responds 200 OK when the process is running.
// A non-200 response causes the container orchestrator to restart the pod.
func (h *HealthHandler) Liveness(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "alive",
	})
}

// Readiness responds 200 OK when the server is ready to serve traffic.
// Currently it always returns ready. Future versions may check DB connectivity.
func (h *HealthHandler) Readiness(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ready",
	})
}
