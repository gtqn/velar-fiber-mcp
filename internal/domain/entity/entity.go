// Package entity defines the core domain entities for VELAR-Fiber.
// These types live at the center of the hexagonal architecture and have
// zero dependencies on any framework or infrastructure library.
package entity

import "time"

// ToolScope defines which API key scopes grant access to a toolset.
type ToolScope string

const (
	// ScopeAll grants access to every toolset.
	ScopeAll ToolScope = "all"
	// ScopeGitHub grants access to the GitHub toolset only.
	ScopeGitHub ToolScope = "github"
	// ScopeDocs grants access to the Docs / Context7 toolset only.
	ScopeDocs ToolScope = "docs"
	// ScopeSystem grants access to the System filesystem toolset only.
	ScopeSystem ToolScope = "system"
	// ScopeWeb grants access to the Web / HTTP toolset only.
	ScopeWeb ToolScope = "web"
	// ScopeUtils grants access to the Utilities toolset only.
	ScopeUtils ToolScope = "utils"
	// ScopeFiber grants access to the specialized Fiber Expert toolset.
	ScopeFiber ToolScope = "fiber"
)

// Toolset groups related tools under a single scope.
type Toolset struct {
	// Name is the unique identifier of the toolset (e.g., "github").
	Name string
	// Scope is the permission required to access this toolset.
	Scope ToolScope
	// Description explains what this toolset does — read by AI agents.
	Description string
}

// APIKey represents an authenticated client with a specific scope.
type APIKey struct {
	// Value is the raw key string (never log this value).
	Value string
	// Scope is the toolset this key can access.
	Scope ToolScope
	// Label is a human-readable name for auditing (e.g., "claude-agent-1").
	Label string
}

// HasAccessTo returns true when the key's scope permits the given toolset scope.
func (k APIKey) HasAccessTo(required ToolScope) bool {
	return k.Scope == ScopeAll || k.Scope == required
}

// SideEffect describes a side effect of executing a tool.
// This metadata helps AI agents reason about safety before calling a tool.
type SideEffect string

const (
	// SideEffectNone means the tool is purely read-only.
	SideEffectNone SideEffect = "none"
	// SideEffectNetworkRead means the tool makes outbound read-only HTTP calls.
	SideEffectNetworkRead SideEffect = "network_read"
	// SideEffectNetworkWrite means the tool performs outbound write operations.
	SideEffectNetworkWrite SideEffect = "network_write"
	// SideEffectFilesystemRead means the tool reads from the local filesystem.
	SideEffectFilesystemRead SideEffect = "filesystem_read"
	// SideEffectFilesystemWrite means the tool writes to the local filesystem.
	SideEffectFilesystemWrite SideEffect = "filesystem_write"
)

// AuditLog represents a single tool execution record for security and monitoring.
type AuditLog struct {
	// TraceID is the unique ID for the entire request chain.
	TraceID string
	// APIKeyLabel is the name of the key that initiated the action.
	APIKeyLabel string
	// ToolName is the fully qualified name of the tool called.
	ToolName string
	// Arguments contains the raw input passed to the tool.
	Arguments any
	// Success indicates if the tool execution was successful.
	Success bool
	// Duration is how long the tool took to execute.
	Duration time.Duration
	// Timestamp is when the action occurred.
	Timestamp time.Time
}

