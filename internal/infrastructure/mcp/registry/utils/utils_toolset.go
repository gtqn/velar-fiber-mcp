// Package utils registers the Utilities toolset on the MCP server.
// All tools in this package are pure functions with no external I/O.
package utils

import (
	"context"
	"crypto/md5"  //nolint:gosec // MD5 here is non-cryptographic use only
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Register adds all Utility tools to the MCP server.
func Register(srv *mcpserver.MCPServer) {
	registerHash(srv)
	registerFormatJSON(srv)
	registerUUID(srv)
	registerTimestamp(srv)
}

// ─── util_hash ────────────────────────────────────────────────────────────────

func registerHash(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"util_hash",
		mcp.WithDescription("Compute a hash of the given text. Supports SHA256 (default) and MD5."),
		mcp.WithString("text", mcp.Required(), mcp.Description("The input text to hash.")),
		mcp.WithString("algorithm",
			mcp.Description("Hash algorithm: 'sha256' (default) or 'md5'."),
			mcp.Enum("sha256", "md5"),
		),
	), handleHash)
}

func handleHash(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := req.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text is required"), nil
	}

	algo := req.GetString("algorithm", "sha256")

	var hash string

	switch algo {
	case "md5":
		sum := md5.Sum([]byte(text)) //nolint:gosec
		hash = hex.EncodeToString(sum[:])
	default:
		sum := sha256.Sum256([]byte(text))
		hash = hex.EncodeToString(sum[:])
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"algorithm":"%s","hash":"%s"}`, algo, hash)), nil
}

// ─── util_format_json ─────────────────────────────────────────────────────────

func registerFormatJSON(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"util_format_json",
		mcp.WithDescription("Pretty-print or minify a JSON string."),
		mcp.WithString("input", mcp.Required(), mcp.Description("The raw JSON string to format.")),
		mcp.WithString("mode",
			mcp.Description("'pretty' (default) adds indentation; 'minify' removes whitespace."),
			mcp.Enum("pretty", "minify"),
		),
	), handleFormatJSON)
}

func handleFormatJSON(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := req.RequireString("input")
	if err != nil {
		return mcp.NewToolResultError("input is required"), nil
	}

	var raw any
	if jsonErr := json.Unmarshal([]byte(input), &raw); jsonErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid JSON: %s", jsonErr.Error())), nil
	}

	mode := req.GetString("mode", "pretty")

	var out []byte
	if mode == "minify" {
		out, _ = json.Marshal(raw)
	} else {
		out, _ = json.MarshalIndent(raw, "", "  ")
	}

	return mcp.NewToolResultText(string(out)), nil
}

// ─── util_uuid ────────────────────────────────────────────────────────────────

func registerUUID(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"util_uuid",
		mcp.WithDescription("Generate one or more UUID v4 values."),
		mcp.WithNumber("count", mcp.Description("Number of UUIDs to generate (1–20, default 1).")),
	), handleUUID)
}

func handleUUID(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// API correct: GetInt(key, default) — NOT GetNumber
	count := req.GetInt("count", 1)
	if count < 1 || count > 20 {
		return mcp.NewToolResultError("count must be between 1 and 20"), nil
	}

	ids := make([]string, count)
	for i := range ids {
		ids[i] = uuid.New().String()
	}

	out, _ := json.Marshal(ids)

	return mcp.NewToolResultText(string(out)), nil
}

// ─── util_timestamp ───────────────────────────────────────────────────────────

func registerTimestamp(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"util_timestamp",
		mcp.WithDescription("Return the current UTC timestamp in multiple formats."),
	), handleTimestamp)
}

func handleTimestamp(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	now := time.Now().UTC()
	out := fmt.Sprintf(
		`{"unix_seconds":%d,"unix_ms":%d,"rfc3339":"%s","iso8601":"%s"}`,
		now.Unix(),
		now.UnixMilli(),
		now.Format(time.RFC3339),
		now.Format("2006-01-02T15:04:05.000Z"),
	)

	return mcp.NewToolResultText(out), nil
}
