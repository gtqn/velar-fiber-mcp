# VELAR-Fiber MCP Server 🚀

An enterprise-grade Model Context Protocol (MCP) server built with **Go** and **Fiber v3**. Designed for high-performance AI agent interactions with strict hexagonal architecture and specialized expertise in the Fiber Framework.

## 🏗️ Architecture
- **Hexagonal (Clean) Architecture**: Strict separation of concerns.
- **Dependency Injection**: Powered by `Uber-Fx`.
- **High Performance**: Built on `Fiber v3` for ultra-low latency.
- **Observability**: Built-in Audit Logging for all AI tool invocations.

## 🛠️ Toolsets

| Toolset | Description | Scopes |
|---|-|---|
| **Fiber Expert** | **Specialized tools for Fiber v3 development, docs, and code gen.** | `fiber` |
| **GitHub** | Search repos, read code, list/create issues, and commit history. | `github` |
| **Docs** | Query documentation via Context7 API. | `docs` |
| **System** | Sandboxed filesystem access (Read/Write/List). | `system` |
| **Web** | Perform HTTP requests and fetch URL content. | `web` |
| **Utils** | JSON formatting, Hashing, UUID generation, and Timestamps. | `utils` |

## 🔐 Security & Auth
The server uses `X-API-Key` authentication.
**Format**: `KEY:SCOPE,KEY2:SCOPE2`
- `SCOPE`: The specific toolset name (e.g., `fiber`, `github`) or `all` for full access.

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- Docker (optional)

### Setup
1. Copy `.env.example` to `.env`.
2. Configure your `API_KEYS` and `GITHUB_TOKEN`.
3. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

### Using with Docker
```bash
docker-compose up --build
```

## 📡 API Endpoints
- `GET /health`: Liveness and Readiness probes.
- `POST /mcp/v1/call`: Bridge endpoint for MCP tool calls (requires Auth).

## 🔌 How to Connect

### 1. Cursor (AI Code Editor)
1. Go to **Cursor Settings** > **Models** > **MCP**.
2. Click **+ Add New MCP Server**.
3. Use the following configuration:
   - **Name**: `VELAR-Fiber`
   - **Type**: `command`
   - **Command**: 
     ```bash
     npx -y mcp-remote http://localhost:8080/mcp/v1 --header "X-API-Key: YOUR_KEY"
     ```

### 2. Claude Desktop
Add this to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "velar-fiber": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "http://localhost:8080/mcp/v1",
        "--header",
        "X-API-Key: YOUR_KEY"
      ]
    }
  }
}
```
> [!TIP]
> Replace `YOUR_KEY` with a valid key from your `.env` file (e.g., `velar-key-1`).

## 📄 License
Enterprise Proprietary.
