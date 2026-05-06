// Package server provides the Fiber HTTP server lifecycle management.
package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/velar/velar-fiber/internal/domain/port"
	"github.com/velar/velar-fiber/internal/infrastructure/config"
	"github.com/velar/velar-fiber/internal/infrastructure/http/handler"
	mw "github.com/velar/velar-fiber/internal/infrastructure/http/middleware"
	"go.uber.org/zap"
)

// FiberServer wraps the Fiber app with lifecycle management.
type FiberServer struct {
	app *fiber.App
	cfg *config.Config
	log *zap.Logger
}

// New creates and configures the Fiber application.
func New(
	cfg *config.Config,
	log *zap.Logger,
	keyRepo port.APIKeyRepository,
	mcpHandler fiber.Handler,
) *FiberServer {
	app := fiber.New(fiber.Config{
		AppName:      cfg.MCP.ServerName + " v" + cfg.MCP.ServerVersion,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: globalErrorHandler(log),
	})

	// Global middlewares
	app.Use(mw.RequestID())
	app.Use(mw.Logger(log))
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "X-API-Key", "X-Request-ID"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
	}))

	health := handler.NewHealthHandler()

	// Health probes — unauthenticated
	app.Get("/health/live", health.Liveness)
	app.Get("/health/ready", health.Readiness)

	// MCP endpoint — authenticated
	mcpGroup := app.Group("/mcp", mw.Auth(keyRepo, log))
	mcpGroup.All("/*", mcpHandler)

	// Root info endpoint
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":    cfg.MCP.ServerName,
			"version": cfg.MCP.ServerVersion,
			"spec":    "2025-11-25",
			"docs":    "https://modelcontextprotocol.io",
		})
	})

	return &FiberServer{app: app, cfg: cfg, log: log}
}

// Start begins listening on the configured port.
func (s *FiberServer) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	s.log.Info("VELAR-Fiber starting",
		zap.String("addr", addr),
		zap.String("env", s.cfg.Server.Env),
	)

	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server within the given context deadline.
func (s *FiberServer) Shutdown(ctx context.Context) error {
	s.log.Info("VELAR-Fiber shutting down gracefully")
	return s.app.ShutdownWithContext(ctx)
}

// globalErrorHandler converts unhandled errors to structured JSON responses.
func globalErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		log.Error("unhandled error", zap.Error(err), zap.String("path", c.Path()))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "an unexpected error occurred",
			},
		})
	}
}
