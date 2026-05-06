// Package system registers the System Filesystem toolset on the MCP server.
// All file operations are sandboxed to the SYSTEM_ALLOWED_PATH directory.
// Attempts to escape the sandbox via path traversal are rejected.
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/mcp"
)

// Toolset holds the sandbox root path — injected at construction time.
type Toolset struct {
	sandboxPath string
}

// New creates a System Toolset scoped to the given sandbox directory.
func New(sandboxPath string) *Toolset {
	return &Toolset{sandboxPath: filepath.Clean(sandboxPath)}
}

// Register adds all System tools to the MCP server.
func (t *Toolset) Register(srv *mcpserver.MCPServer) {
	t.registerReadFile(srv)
	t.registerWriteFile(srv)
	t.registerListDir(srv)
	t.registerGetEnv(srv)
}

// safeJoin joins the sandbox root with a relative path and verifies
// the resulting path is still inside the sandbox.
func (t *Toolset) safeJoin(rel string) (string, error) {
	abs := filepath.Clean(filepath.Join(t.sandboxPath, rel))
	if !strings.HasPrefix(abs, t.sandboxPath) {
		return "", fmt.Errorf("path traversal detected: %q is outside the sandbox", rel)
	}

	return abs, nil
}

// ─── sys_read_file ────────────────────────────────────────────────────────────

func (t *Toolset) registerReadFile(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"sys_read_file",
		mcp.WithDescription("Read the text content of a file within the sandbox directory. Side effect: FILESYSTEM_READ."),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Relative path to the file within the sandbox (e.g., 'data/input.json')."),
		),
	), t.handleReadFile)
}

func (t *Toolset) handleReadFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rel, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path is required"), nil
	}

	abs, err := t.safeJoin(rel)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, err := os.ReadFile(abs) //nolint:gosec // path is sandboxed
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot read file: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// ─── sys_write_file ───────────────────────────────────────────────────────────

func (t *Toolset) registerWriteFile(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"sys_write_file",
		mcp.WithDescription("Write text content to a file within the sandbox directory. Creates parent directories as needed. Side effect: FILESYSTEM_WRITE."),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Relative path to the file within the sandbox."),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The text content to write to the file."),
		),
	), t.handleWriteFile)
}

func (t *Toolset) handleWriteFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rel, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path is required"), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content is required"), nil
	}

	abs, err := t.safeJoin(rel)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot create directories: %s", err.Error())), nil
	}

	if err := os.WriteFile(abs, []byte(content), 0o600); err != nil { //nolint:gosec
		return mcp.NewToolResultError(fmt.Sprintf("cannot write file: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"written":true,"path":"%s"}`, rel)), nil
}

// ─── sys_list_directory ───────────────────────────────────────────────────────

func (t *Toolset) registerListDir(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"sys_list_directory",
		mcp.WithDescription("List files and directories at a path within the sandbox. Side effect: FILESYSTEM_READ."),
		mcp.WithString("path",
			mcp.Description("Relative path within the sandbox. Defaults to sandbox root."),
		),
	), t.handleListDir)
}

func (t *Toolset) handleListDir(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rel := req.GetString("path", ".")

	abs, err := t.safeJoin(rel)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot list directory: %s", err.Error())), nil
	}

	type entry struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
	}

	items := make([]entry, 0, len(entries))
	for _, e := range entries {
		items = append(items, entry{Name: e.Name(), IsDir: e.IsDir()})
	}

	out, _ := json.Marshal(items)

	return mcp.NewToolResultText(string(out)), nil
}

// allowedEnvVars is the whitelist of environment variables AI agents may read.
// This prevents accidental exposure of secrets like GITHUB_TOKEN.
var allowedEnvVars = map[string]bool{ //nolint:gochecknoglobals
	"MCP_SERVER_NAME":    true,
	"MCP_SERVER_VERSION": true,
	"SERVER_ENV":         true,
	"OTEL_SERVICE_NAME":  true,
}

// ─── sys_get_env ──────────────────────────────────────────────────────────────

func (t *Toolset) registerGetEnv(srv *mcpserver.MCPServer) {
	srv.AddTool(mcp.NewTool(
		"sys_get_env",
		mcp.WithDescription("Read a whitelisted environment variable. Only safe, non-sensitive variables are accessible. Side effect: none."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The environment variable name (e.g., SERVER_ENV, MCP_SERVER_NAME)."),
		),
	), t.handleGetEnv)
}

func (t *Toolset) handleGetEnv(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}

	if !allowedEnvVars[name] {
		return mcp.NewToolResultError(fmt.Sprintf(
			"variable %q is not in the whitelist. Allowed: MCP_SERVER_NAME, MCP_SERVER_VERSION, SERVER_ENV, OTEL_SERVICE_NAME", name,
		)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"%s":"%s"}`, name, os.Getenv(name))), nil
}
