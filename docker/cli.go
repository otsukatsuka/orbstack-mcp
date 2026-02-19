package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CLI implements Executor by shelling out to the docker binary.
type CLI struct{}

func NewCLI() *CLI {
	return &CLI{}
}

func (c *CLI) Exec(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", args[0], err, stderr.String())
	}
	return stdout.String(), nil
}

func (c *CLI) ExecCombined(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %w: %s", args[0], err, combined.String())
	}
	return combined.String(), nil
}
