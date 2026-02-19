package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleComposeLogs_TwoServices(t *testing.T) {
	mock := docker.NewMock()

	// Register ps output with two containers.
	psOutput := `{"ID":"abc123","Names":"myapp-web-1","Labels":"com.docker.compose.project=myapp,com.docker.compose.service=web"}
{"ID":"def456","Names":"myapp-db-1","Labels":"com.docker.compose.project=myapp,com.docker.compose.service=db"}`
	mock.On("ps -a --format {{json .}} --filter label=com.docker.compose.project=myapp", psOutput, nil)

	// Register logs output for each container.
	mock.On("logs --tail 100 abc123", "web log line 1\nweb log line 2\n", nil)
	mock.On("logs --tail 100 def456", "db log line 1\ndb log line 2\n", nil)

	result, err := handleComposeLogs(context.Background(), mock, composeLogsArgs{
		Project: "myapp",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that each service log line is prefixed.
	if !strings.Contains(result, "[web] web log line 1") {
		t.Errorf("expected web log line 1, got: %s", result)
	}
	if !strings.Contains(result, "[web] web log line 2") {
		t.Errorf("expected web log line 2, got: %s", result)
	}
	if !strings.Contains(result, "[db] db log line 1") {
		t.Errorf("expected db log line 1, got: %s", result)
	}
	if !strings.Contains(result, "[db] db log line 2") {
		t.Errorf("expected db log line 2, got: %s", result)
	}
}

func TestHandleComposeLogs_ProjectNotFound(t *testing.T) {
	mock := docker.NewMock()

	// Return empty output for ps - no containers found.
	mock.On("ps -a --format {{json .}} --filter label=com.docker.compose.project=nonexistent", "", nil)

	_, err := handleComposeLogs(context.Background(), mock, composeLogsArgs{
		Project: "nonexistent",
	})

	if err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
	if !strings.Contains(err.Error(), "no containers found") {
		t.Errorf("expected 'no containers found' error, got: %v", err)
	}
}

func TestHandleComposeLogs_DockerError(t *testing.T) {
	mock := docker.NewMock()

	// Simulate docker ps failure.
	mock.On("ps -a --format {{json .}} --filter label=com.docker.compose.project=myapp", "", fmt.Errorf("docker daemon not running"))

	_, err := handleComposeLogs(context.Background(), mock, composeLogsArgs{
		Project: "myapp",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list containers") {
		t.Errorf("expected 'failed to list containers' error, got: %v", err)
	}
}

func TestHandleComposeLogs_WithOptions(t *testing.T) {
	mock := docker.NewMock()

	psOutput := `{"ID":"abc123","Names":"myapp-web-1","Labels":"com.docker.compose.project=myapp,com.docker.compose.service=web"}`
	mock.On("ps -a --format {{json .}} --filter label=com.docker.compose.project=myapp", psOutput, nil)

	mock.On("logs --tail 50 --since 1h --timestamps abc123", "2024-01-01T00:00:00Z log line\n", nil)

	result, err := handleComposeLogs(context.Background(), mock, composeLogsArgs{
		Project:    "myapp",
		Tail:       50,
		Since:      "1h",
		Timestamps: true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[web] 2024-01-01T00:00:00Z log line") {
		t.Errorf("expected timestamped log line, got: %s", result)
	}
}
