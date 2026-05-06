// Package web registers the Web & HTTP toolset on the MCP server.
// These tools allow AI agents to make outbound HTTP requests and fetch
// web content — all calls go through a shared client with timeout enforcement.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/mcp"
)

// sharedClient is the single HTTP client used by all web tools.
// Using a shared client enables connection pooling.
var sharedClient = &http.Client{ //nolint:gochecknoglobals
	Timeout: 30 * time.Second,
}

// Register adds all Web tools to the MCP server.
func Register(srv *mcpserver.MCPServer) {
	registerHTTPRequest(srv)
	registerFetchURL(srv)
}

// ─── http_request ─────────────────────────────────────────────────────────────

func registerHTTPRequest(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"http_request",
		mcp.WithDescription("Make an outbound HTTP request. Supports GET, POST, PUT, PATCH, DELETE. Returns status code, headers, and body. Side effect: NETWORK_WRITE (for non-GET methods)."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The full URL to request (must start with https:// or http://)."),
		),
		mcp.WithString("method",
			mcp.Description("HTTP method: GET (default), POST, PUT, PATCH, DELETE."),
			mcp.Enum("GET", "POST", "PUT", "PATCH", "DELETE"),
		),
		mcp.WithString("body",
			mcp.Description("Request body as a string (for POST/PUT/PATCH)."),
		),
		mcp.WithString("content_type",
			mcp.Description("Content-Type header (default: application/json)."),
		),
		mcp.WithString("headers",
			mcp.Description("Additional headers as a JSON object: {\"Authorization\": \"Bearer token\"}."),
		),
	), handleHTTPRequest)
}

func handleHTTPRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil || rawURL == "" {
		return mcp.NewToolResultError("url is required and must not be empty"), nil
	}

	method := strings.ToUpper(req.GetString("method", "GET"))
	body := req.GetString("body", "")
	contentType := req.GetString("content_type", "application/json")
	headersJSON := req.GetString("headers", "")

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, rawURL, reqBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build request: %s", err.Error())), nil
	}

	if body != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// Apply additional headers.
	if headersJSON != "" {
		var extra map[string]string
		if err := json.Unmarshal([]byte(headersJSON), &extra); err == nil {
			for k, v := range extra {
				httpReq.Header.Set(k, v)
			}
		}
	}

	resp, err := sharedClient.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("request failed: %s", err.Error())), nil
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return mcp.NewToolResultError("failed to read response body"), nil
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
	}

	out, _ := json.Marshal(result)

	return mcp.NewToolResultText(string(out)), nil
}

// ─── web_fetch_url ────────────────────────────────────────────────────────────

func registerFetchURL(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"web_fetch_url",
		mcp.WithDescription("Fetch the text content of a web page. Returns the raw HTML (or plain text if the server sends it). Useful for reading documentation, changelogs, or API specs. Side effect: NETWORK_READ."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to fetch."),
		),
	), handleFetchURL)
}

func handleFetchURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError("url is required"), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid URL: %s", err.Error())), nil
	}

	httpReq.Header.Set("User-Agent", "VELAR-Fiber/1.0 (MCP Server; +https://github.com/velar/velar-fiber)")

	resp, err := sharedClient.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("fetch failed: %s", err.Error())), nil
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MB limit
	if err != nil {
		return mcp.NewToolResultError("failed to read page content"), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}
