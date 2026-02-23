# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

orbstack-mcp is a Go-based MCP (Model Context Protocol) server for OrbStack that enables AI assistants to operate, monitor, and debug Docker containers. It exposes 12 tools via the MCP stdio transport, built on the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) (`mcp` package).

## Build & Test Commands

```bash
go build ./...                    # Build all packages
go build -o orbstack-mcp .       # Build binary
go test ./... -v                  # Run all tests
go test ./tools -v -run TestName # Run a single test
```

Go version: 1.25+ (go.mod), toolchain managed via `.tool-versions`.

## Architecture

Three-layer design:

```
main.go → mcp.Server (stdio transport)
  ↓
tools/  → 12 tool handlers + registration
  ↓
docker/ → Executor interface (CLI impl + Mock for tests)
```

**Entry point** (`main.go`): Creates `mcp.Server`, injects `docker.CLI` executor, registers all tools via `tools.RegisterAll()`, runs stdio transport.

**`docker/` package**: Defines the `Executor` interface with `Exec` (stdout) and `ExecCombined` (stdout+stderr) methods. `CLI` is the real implementation using `exec.CommandContext`. `Mock` is for tests — register expected command→output pairs with `mock.On("args", output, err)`.

**`tools/` package**: Each tool file follows a consistent pattern:
1. `xxxArgs` struct with `json` + `jsonschema` tags (schema auto-generated from struct tags)
2. `handleXxx(ctx, exec, args) (string, error)` — pure logic, testable via mock executor
3. `registerXxx(server, exec)` — wires handler into MCP server via `mcp.AddTool`

All tools return text (not structured data). Errors are returned as `CallToolResult` with `IsError: true`, not as Go errors.

## Adding a New Tool

1. Create `tools/new_tool.go` following the args struct → handler → register pattern
2. Create `tools/new_tool_test.go` using `docker.NewMock()` and `mock.On()`
3. Add `registerNewTool(server, exec)` call in `tools/register.go`

## Key Design Decisions

- **Docker commands use `exec.CommandContext` with explicit args** — no shell interpolation, safe from injection
- **Compose project discovery**: Tools find working directories via `com.docker.compose.project.working_dir` container label
- **`ExecCombined` for logs**: `docker logs` writes to stderr, so log-related tools use `ExecCombined`
- **Mock keyed by joined args**: `docker.Mock` uses `strings.Join(args, " ")` as lookup key — test registrations must match exact arg strings
