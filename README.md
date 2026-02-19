# orbstack-mcp

MCP (Model Context Protocol) server for [OrbStack](https://orbstack.dev/) that enables AI assistants like Claude Code and Cursor to directly operate, monitor, and debug Docker containers.

Built with Go using the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).

## Features

- **12 tools** for comprehensive container management
- Works with OrbStack's Docker runtime out of the box
- Compose project awareness (grouping, project-level operations)
- Safe execution: no shell injection, commands run via `exec.CommandContext`
- Stdio transport for seamless integration with MCP clients

## Installation

```bash
go install github.com/otsukatsuka/orbstack-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/otsukatsuka/orbstack-mcp.git
cd orbstack-mcp
go build -o orbstack-mcp .
```

## Configuration

### Claude Code

Add to your Claude Code MCP settings (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "orbstack": {
      "command": "orbstack-mcp"
    }
  }
}
```

Or if built from source:

```json
{
  "mcpServers": {
    "orbstack": {
      "command": "/path/to/orbstack-mcp"
    }
  }
}
```

### Cursor

Add to your Cursor MCP settings (`.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "orbstack": {
      "command": "orbstack-mcp"
    }
  }
}
```

## Tools

### Core

| Tool | Description |
|------|-------------|
| `list_containers` | List containers grouped by Compose project. Supports filtering by project name. |
| `get_logs` | Get container logs with tail/since/until/timestamps options. |
| `search_logs` | Search logs with regex patterns. Supports context lines (like `grep -C`). |
| `compose_logs` | Get merged logs for all services in a Compose project, prefixed with service names. |

### Debug

| Tool | Description |
|------|-------------|
| `container_exec` | Execute commands inside a container via `sh -c` (supports pipes/redirects). |
| `restart_service` | Restart a container with configurable timeout. |
| `container_stats` | Get CPU/memory/network/block I/O statistics snapshot. |

### Inspect & Troubleshoot

| Tool | Description |
|------|-------------|
| `container_inspect` | Get detailed container info with section filtering (env/ports/volumes/network/all). |
| `container_health` | Get health check configuration and recent check results. |
| `log_diff` | Compare logs between two time periods for regression debugging. |

### Compose & Events

| Tool | Description |
|------|-------------|
| `compose_up` | Start a Compose project (auto-discovers working directory from containers). |
| `compose_down` | Stop a Compose project with optional volume removal. |
| `container_events` | Get container event history (start/stop/die/restart/OOM). |

## Development

### Prerequisites

- Go 1.24+
- Docker (via OrbStack)

### Run tests

```bash
go test ./... -v
```

### Build

```bash
go build ./...
```

## License

Apache 2.0 - see [LICENSE](LICENSE).
