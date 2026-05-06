// Package di wires the entire application using Uber-Fx dependency injection.
// This is the ONLY place where concrete types are instantiated and connected.
// Every other package depends only on interfaces — never on other concrete types.
package di

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/velar/velar-fiber/internal/domain/port"
	"github.com/velar/velar-fiber/internal/infrastructure/config"
	httpserver "github.com/velar/velar-fiber/internal/infrastructure/http"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/docs"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/fiber"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/github"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/system"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/utils"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/web"
	"github.com/velar/velar-fiber/internal/infrastructure/persistence/memory"
	pkglogger "github.com/velar/velar-fiber/pkg/logger"
)

// provideLogger builds the Zap logger from config.
func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	return pkglogger.New(cfg.Server.Env)
}

// provideAPIKeyRepo builds the in-memory API key repository.
func provideAPIKeyRepo(cfg *config.Config) (port.APIKeyRepository, error) {
	return memory.NewAPIKeyMemoryRepo(cfg.Auth.APIKeysRaw)
}

// provideAuditRepo builds the audit logging repository.
func provideAuditRepo(log *zap.Logger) port.AuditLogger {
	return memory.NewAuditMemoryRepo(log)
}

// provideGitHubToolset constructs the GitHub toolset with token injection.
func provideGitHubToolset(cfg *config.Config) *github.Toolset {
	return github.New(cfg.GitHub.Token, cfg.GitHub.Host)
}

// provideDocsToolset constructs the Docs/Context7 toolset.
func provideDocsToolset(cfg *config.Config) *docs.Toolset {
	return docs.New(cfg.Context7.BaseURL, cfg.Context7.APIKey)
}

// provideSystemToolset constructs the System toolset with its sandbox path.
func provideSystemToolset(cfg *config.Config) *system.Toolset {
	return system.New(cfg.System.AllowedPath)
}

// provideFiberExpert constructs the specialized Fiber Framework Expert.
func provideFiberExpert(gh *github.Toolset, dc *docs.Toolset) *fiber.Toolset {
	return fiber.New(gh, dc)
}

// provideMCPBridge builds the MCP server and registers all toolsets.
func provideMCPBridge(
	cfg *config.Config,
	log *zap.Logger,
	audit port.AuditLogger,
	gh *github.Toolset,
	dc *docs.Toolset,
	sys *system.Toolset,
	fib *fiber.Toolset,
) *mcp.Bridge {
	return mcp.NewBridge(
		cfg.MCP.ServerName,
		cfg.MCP.ServerVersion,
		log,
		audit,
		utils.Register,
		web.Register,
		sys.Register,
		dc.Register,
		gh.Register,
		fib.Register,
	)
}

// provideFiberServer wires the MCP bridge into the Fiber HTTP server.
func provideFiberServer(
	cfg *config.Config,
	log *zap.Logger,
	keyRepo port.APIKeyRepository,
	bridge *mcp.Bridge,
) *httpserver.FiberServer {
	return httpserver.New(cfg, log, keyRepo, bridge.FiberHandler())
}

// startServer registers the server lifecycle with Uber-Fx.
func startServer(lc fx.Lifecycle, srv *httpserver.FiberServer, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := srv.Start(); err != nil {
					log.Error("server error", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

// Run bootstraps and starts the VELAR-Fiber application.
func Run() {
	app := fx.New(
		fx.Provide(
			config.Load,
			provideLogger,
			provideAPIKeyRepo,
			provideAuditRepo,
			provideGitHubToolset,
			provideDocsToolset,
			provideSystemToolset,
			provideFiberExpert,
			provideMCPBridge,
			provideFiberServer,
		),
		fx.Invoke(startServer),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if err := app.Stop(stopCtx); err != nil {
		os.Exit(1)
	}
}
