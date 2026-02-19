package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
	"github.com/otsukatsuka/orbstack-mcp/tools"
)

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "orbstack-mcp",
			Version: "0.1.0",
		},
		nil,
	)

	exec := docker.NewCLI()
	tools.RegisterAll(server, exec)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
