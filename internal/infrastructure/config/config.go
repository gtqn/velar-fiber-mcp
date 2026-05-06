// Package config loads and validates all application configuration from
// environment variables. It uses Viper under the hood but exposes a clean
// typed struct so no other package ever imports Viper directly.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the single source of truth for all runtime configuration.
// It is injected via Uber-Fx — never accessed as a global.
type Config struct {
	Server  ServerConfig
	MCP     MCPConfig
	Auth    AuthConfig
	GitHub  GitHubConfig
	Context7 Context7Config
	System  SystemConfig
	OTEL    OTELConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int    `mapstructure:"SERVER_PORT"`
	Env  string `mapstructure:"SERVER_ENV"`
}

// MCPConfig holds Model Context Protocol server settings.
type MCPConfig struct {
	ServerName          string `mapstructure:"MCP_SERVER_NAME"`
	ServerVersion       string `mapstructure:"MCP_SERVER_VERSION"`
	MaxConcurrentTasks  int    `mapstructure:"MCP_MAX_CONCURRENT_TASKS"`
}

// AuthConfig holds API key authentication settings.
type AuthConfig struct {
	// APIKeys is the raw string: "key1:scope1,key2:scope2"
	APIKeysRaw  string `mapstructure:"API_KEYS"`
	RateLimitRPM int   `mapstructure:"RATE_LIMIT_RPM"`
}

// GitHubConfig holds credentials for the GitHub toolset.
type GitHubConfig struct {
	Token string `mapstructure:"GITHUB_TOKEN"`
	Host  string `mapstructure:"GITHUB_HOST"`
}

// Context7Config holds credentials for the Docs toolset.
type Context7Config struct {
	APIKey  string `mapstructure:"CONTEXT7_API_KEY"`
	BaseURL string `mapstructure:"CONTEXT7_BASE_URL"`
}

// SystemConfig holds sandbox settings for the System toolset.
type SystemConfig struct {
	AllowedPath string `mapstructure:"SYSTEM_ALLOWED_PATH"`
}

// OTELConfig holds OpenTelemetry settings.
type OTELConfig struct {
	Enabled     bool   `mapstructure:"OTEL_ENABLED"`
	Endpoint    string `mapstructure:"OTEL_ENDPOINT"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME"`
}

// Load reads configuration from environment variables and .env file.
// Returns an error if any required field is missing.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_ENV", "development")
	v.SetDefault("MCP_SERVER_NAME", "VELAR-Fiber")
	v.SetDefault("MCP_SERVER_VERSION", "1.0.0")
	v.SetDefault("MCP_MAX_CONCURRENT_TASKS", 50)
	v.SetDefault("RATE_LIMIT_RPM", 100)
	v.SetDefault("GITHUB_HOST", "https://api.github.com")
	v.SetDefault("CONTEXT7_BASE_URL", "https://mcp.context7.com")
	v.SetDefault("SYSTEM_ALLOWED_PATH", "/tmp/velar-sandbox")
	v.SetDefault("OTEL_ENABLED", false)
	v.SetDefault("OTEL_SERVICE_NAME", "velar-fiber")

	// Read .env file if present (non-fatal if missing)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	// Environment variables always override .env
	v.AutomaticEnv()

	cfg := &Config{}
	if err := bindAll(v, cfg); err != nil {
		return nil, fmt.Errorf("binding config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// bindAll maps viper keys to the struct fields.
func bindAll(v *viper.Viper, cfg *Config) error {
	cfg.Server.Port = v.GetInt("SERVER_PORT")
	cfg.Server.Env = v.GetString("SERVER_ENV")

	cfg.MCP.ServerName = v.GetString("MCP_SERVER_NAME")
	cfg.MCP.ServerVersion = v.GetString("MCP_SERVER_VERSION")
	cfg.MCP.MaxConcurrentTasks = v.GetInt("MCP_MAX_CONCURRENT_TASKS")

	cfg.Auth.APIKeysRaw = v.GetString("API_KEYS")
	cfg.Auth.RateLimitRPM = v.GetInt("RATE_LIMIT_RPM")

	cfg.GitHub.Token = v.GetString("GITHUB_TOKEN")
	cfg.GitHub.Host = v.GetString("GITHUB_HOST")

	cfg.Context7.APIKey = v.GetString("CONTEXT7_API_KEY")
	cfg.Context7.BaseURL = v.GetString("CONTEXT7_BASE_URL")

	cfg.System.AllowedPath = v.GetString("SYSTEM_ALLOWED_PATH")

	cfg.OTEL.Enabled = v.GetBool("OTEL_ENABLED")
	cfg.OTEL.Endpoint = v.GetString("OTEL_ENDPOINT")
	cfg.OTEL.ServiceName = v.GetString("OTEL_SERVICE_NAME")

	return nil
}

// validate checks that critical fields are non-empty.
func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.Auth.APIKeysRaw) == "" {
		return fmt.Errorf("API_KEYS is required — set at least one key:scope pair")
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	return nil
}
