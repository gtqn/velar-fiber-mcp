// Package docs registers the Docs toolset on the MCP server.
// It integrates with Context7 (https://mcp.context7.com) to provide
// up-to-date library documentation for AI agents.
package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/mcp"
)

// Toolset holds the Context7 client configuration.
type Toolset struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New creates a Docs Toolset backed by the Context7 API.
func New(baseURL, apiKey string) *Toolset {
	return &Toolset{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

// Register adds all Docs tools to the MCP server.
func (t *Toolset) Register(srv *mcpserver.MCPServer) {
	t.registerResolveLibrary(srv)
	t.registerQueryDocs(srv)
	t.registerSearchWeb(srv)
}

// ─── docs_resolve_library ─────────────────────────────────────────────────────

func (t *Toolset) registerResolveLibrary(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"docs_resolve_library",
		mcp.WithDescription("Resolve a library name to a Context7-compatible library ID. Call this FIRST before docs_query_library to get the exact ID. Side effect: NETWORK_READ."),
		mcp.WithString("library_name",
			mcp.Required(),
			mcp.Description("The name of the library to look up (e.g., 'Next.js', 'Fiber', 'React')."),
		),
		mcp.WithString("query",
			mcp.Description("Optional context about what you want to do with the library (improves relevance ranking)."),
		),
	), t.handleResolveLibrary)
}

func (t *Toolset) handleResolveLibrary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("library_name")
	if err != nil {
		return mcp.NewToolResultError("library_name is required"), nil
	}

	query := req.GetString("query", name)

	// Context7 resolve endpoint: /v1/libraries?query=...&libraryName=...
	endpoint := fmt.Sprintf("%s/v1/libraries?libraryName=%s&query=%s",
		t.baseURL,
		url.QueryEscape(name),
		url.QueryEscape(query),
	)

	body, err := t.get(ctx, endpoint)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Context7 resolve failed: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

// ─── docs_query_library ───────────────────────────────────────────────────────

func (t *Toolset) registerQueryDocs(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"docs_query_library",
		mcp.WithDescription("Retrieve up-to-date documentation and code examples for a library using its Context7 ID. Use docs_resolve_library first to get the ID. Side effect: NETWORK_READ."),
		mcp.WithString("library_id",
			mcp.Required(),
			mcp.Description("Context7-compatible library ID (e.g., '/vercel/next.js', '/gofiber/fiber')."),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("What you need to know (e.g., 'how to set up middleware in Fiber v3')."),
		),
		mcp.WithNumber("tokens",
			mcp.Description("Maximum number of tokens to return (default 5000, max 10000)."),
		),
	), t.handleQueryDocs)
}

func (t *Toolset) handleQueryDocs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	libraryID, err := req.RequireString("library_id")
	if err != nil {
		return mcp.NewToolResultError("library_id is required"), nil
	}

	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	tokens := req.GetInt("tokens", 5000)
	return t.HandleQuery(ctx, libraryID, query, tokens)
}

// HandleQuery is the internal engine for querying Context7. 
// Exported so other specialized toolsets (like FiberExpert) can reuse it.
func (t *Toolset) HandleQuery(ctx context.Context, libraryID, query string, tokens int) (*mcp.CallToolResult, error) {
	if tokens > 10000 {
		tokens = 10000
	}

	// Remove leading slash if present (e.g., /gofiber/fiber -> gofiber/fiber)
	libraryID = strings.TrimPrefix(libraryID, "/")

	// Context7 Docs endpoint: /v1/{vendor}/{project}?query=...&tokens=...
	endpoint := fmt.Sprintf("%s/v1/%s?query=%s&tokens=%d",
		t.baseURL,
		libraryID, // No QueryEscape here because it contains the slash for vendor/project
		url.QueryEscape(query),
		tokens,
	)

	body, err := t.get(ctx, endpoint)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Context7 docs query failed: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

// ─── docs_search_web ─────────────────────────────────────────────────────────

func (t *Toolset) registerSearchWeb(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"docs_search_web",
		mcp.WithDescription("Search the web for documentation, articles, or answers. Returns structured results with titles and URLs. Side effect: NETWORK_READ."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query (e.g., 'Fiber v3 middleware best practices')."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (1–10, default 5)."),
		),
	), t.handleSearchWeb)
}

func (t *Toolset) handleSearchWeb(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query is required"), nil
	}

	limit := req.GetInt("limit", 5)
	if limit < 1 || limit > 10 {
		limit = 5
	}

	// Use DuckDuckGo Lite as a simple, unauthenticated search backend.
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	body, err := t.get(ctx, searchURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %s", err.Error())), nil
	}

	// Return raw HTML truncated to a manageable size — AI can parse it.
	maxLen := 4096 * limit
	if len(body) > maxLen {
		body = body[:maxLen]
	}

	result := map[string]any{
		"query":   query,
		"results": string(body),
	}

	out, _ := json.Marshal(result)

	return mcp.NewToolResultText(string(out)), nil
}

// get performs an authenticated GET request to the given endpoint.
func (t *Toolset) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if t.apiKey != "" {
		req.Header.Set("X-Context7-API-Key", t.apiKey)
	}

	req.Header.Set("User-Agent", "VELAR-Fiber/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}
