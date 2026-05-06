// Package fiber registers the specialized Fiber Framework Expert toolset.
package fiber

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/docs"
	"github.com/velar/velar-fiber/internal/infrastructure/mcp/registry/github"
)

// Toolset is the specialized expert for the Fiber framework.
// It leverages the general Docs and GitHub toolsets internally.
type Toolset struct {
	gh *github.Toolset
	dc *docs.Toolset
}

// New creates a new Fiber Expert toolset.
func New(gh *github.Toolset, dc *docs.Toolset) *Toolset {
	return &Toolset{gh: gh, dc: dc}
}

// Register adds all Fiber Expert tools to the MCP server.
func (t *Toolset) Register(srv *mcpserver.MCPServer) {
	t.registerSearchDocs(srv)
	t.registerTrackReleases(srv)
	t.registerGenerateBoilerplate(srv)
}

// ─── fiber_search_docs ───────────────────────────────────────────────────────

func (t *Toolset) registerSearchDocs(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"fiber_search_docs",
		mcp.WithDescription("Expert search for Fiber v3 documentation. Automatically targets the latest framework standards. Side effect: NETWORK_READ."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Specific feature or issue to search in Fiber docs (e.g., 'v3 context cleanup').")),
	), t.handleSearchDocs)
}

func (t *Toolset) handleSearchDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, _ := req.RequireString("query")
	
	// Pre-configured Fiber v3 Library ID for Context7
	fiberLibID := "/gofiber/docs" 
	
	// We call the Docs toolset logic (via its registered handler or exported method)
	// For simplicity, we implement the call here since we have the dc client.
	return t.dc.HandleQuery(ctx, fiberLibID, query, 5000)
}

// ─── fiber_track_releases ────────────────────────────────────────────────────

func (t *Toolset) registerTrackReleases(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"fiber_track_releases",
		mcp.WithDescription("Get latest stable releases and breaking changes from Fiber's official repository. Side effect: NETWORK_READ."),
	), t.handleTrackReleases)
}

func (t *Toolset) handleTrackReleases(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Reusing the GitHub toolset logic to fetch commits/tags
	// We can directly use the gh toolset's logic here.
	return t.gh.HandleListCommits(ctx, "gofiber", "fiber", "main", 10)
}

// ─── fiber_generate_boilerplate ──────────────────────────────────────────────

func (t *Toolset) registerGenerateBoilerplate(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"fiber_generate_boilerplate",
		mcp.WithDescription("Generates production-ready Fiber v3 code snippets for common tasks."),
		mcp.WithString("template", mcp.Required(), 
			mcp.Description("Template type: 'server', 'middleware', 'group', 'ws'."),
			mcp.Enum("server", "middleware", "group", "ws"),
		),
	), t.handleGenerateBoilerplate)
}

func (t *Toolset) handleGenerateBoilerplate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tpl, _ := req.RequireString("template")

	var code string
	switch tpl {
	case "server":
		code = `app := fiber.New(fiber.Config{
    AppName: "Velar App",
})
app.Get("/", func(c fiber.Ctx) error {
    return c.SendString("Fiber v3 is live!")
})
app.Listen(":3000")`
	case "middleware":
		code = `func NewAuthMiddleware() fiber.Handler {
    return func(c fiber.Ctx) error {
        token := c.Get("X-API-Key")
        if token == "" {
            return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
        }
        return c.Next()
    }
}`
	case "group":
		code = `api := app.Group("/api/v1")
v1.Get("/users", userHandler)
v1.Post("/data", dataHandler)`
	case "ws":
		code = `app.Get("/ws", websocket.New(func(c *websocket.Conn) {
    for {
        mt, msg, _ := c.ReadMessage()
        c.WriteMessage(mt, msg)
    }
}))`
	}

	return mcp.NewToolResultText(fmt.Sprintf("```go\n%s\n```", code)), nil
}
