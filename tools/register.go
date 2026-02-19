package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

// RegisterAll registers all OrbStack MCP tools on the server.
func RegisterAll(server *mcp.Server, exec docker.Executor) {
	registerListContainers(server, exec)
	registerGetLogs(server, exec)
	registerSearchLogs(server, exec)
	registerComposeLogs(server, exec)
	registerContainerExec(server, exec)
	registerRestartService(server, exec)
	registerContainerStats(server, exec)
	registerContainerInspect(server, exec)
	registerContainerHealth(server, exec)
	registerLogDiff(server, exec)
	registerComposeUpDown(server, exec)
	registerContainerEvents(server, exec)
}
