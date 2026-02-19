package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleContainerHealth_Healthy(t *testing.T) {
	mock := docker.NewMock()

	healthJSON := `{"Status":"healthy","FailingStreak":0,"Log":[{"Start":"2024-01-01T00:00:00Z","End":"2024-01-01T00:00:01Z","ExitCode":0,"Output":"OK"},{"Start":"2024-01-01T00:01:00Z","End":"2024-01-01T00:01:01Z","ExitCode":0,"Output":"OK"}]}`
	configJSON := `{"Test":["CMD-SHELL","curl -f http://localhost/ || exit 1"],"Interval":30000000000,"Timeout":10000000000,"Retries":3}`

	mock.On("inspect --format {{json .State.Health}} myapp", healthJSON, nil)
	mock.On("inspect --format {{json .Config.Healthcheck}} myapp", configJSON, nil)

	args := containerHealthArgs{Container: "myapp"}
	result, err := handleContainerHealth(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "healthy") {
		t.Errorf("expected 'healthy' status, got:\n%s", result)
	}
	if !strings.Contains(result, "Failing Streak: 0") {
		t.Errorf("expected 'Failing Streak: 0', got:\n%s", result)
	}
	if !strings.Contains(result, "CMD-SHELL") {
		t.Errorf("expected health check command, got:\n%s", result)
	}
	if !strings.Contains(result, "curl") {
		t.Errorf("expected curl command in test, got:\n%s", result)
	}
	if !strings.Contains(result, "30s") {
		t.Errorf("expected interval '30s', got:\n%s", result)
	}
	if !strings.Contains(result, "10s") {
		t.Errorf("expected timeout '10s', got:\n%s", result)
	}
	if !strings.Contains(result, "Retries:  3") {
		t.Errorf("expected 'Retries:  3', got:\n%s", result)
	}
	if !strings.Contains(result, "2 entries") {
		t.Errorf("expected '2 entries' in log section, got:\n%s", result)
	}
	if !strings.Contains(result, "Exit Code: 0") {
		t.Errorf("expected 'Exit Code: 0', got:\n%s", result)
	}
}

func TestHandleContainerHealth_Unhealthy(t *testing.T) {
	mock := docker.NewMock()

	healthJSON := `{"Status":"unhealthy","FailingStreak":5,"Log":[{"Start":"2024-01-01T00:00:00Z","End":"2024-01-01T00:00:01Z","ExitCode":1,"Output":"Connection refused"}]}`
	configJSON := `{"Test":["CMD-SHELL","curl -f http://localhost/ || exit 1"],"Interval":30000000000,"Timeout":10000000000,"Retries":3}`

	mock.On("inspect --format {{json .State.Health}} sickapp", healthJSON, nil)
	mock.On("inspect --format {{json .Config.Healthcheck}} sickapp", configJSON, nil)

	args := containerHealthArgs{Container: "sickapp"}
	result, err := handleContainerHealth(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "unhealthy") {
		t.Errorf("expected 'unhealthy' status, got:\n%s", result)
	}
	if !strings.Contains(result, "Failing Streak: 5") {
		t.Errorf("expected 'Failing Streak: 5', got:\n%s", result)
	}
	if !strings.Contains(result, "Exit Code: 1") {
		t.Errorf("expected 'Exit Code: 1', got:\n%s", result)
	}
	if !strings.Contains(result, "Connection refused") {
		t.Errorf("expected 'Connection refused' output, got:\n%s", result)
	}
}

func TestHandleContainerHealth_NoHealthcheck(t *testing.T) {
	mock := docker.NewMock()

	mock.On("inspect --format {{json .State.Health}} nocheck", "null", nil)
	mock.On("inspect --format {{json .Config.Healthcheck}} nocheck", "null", nil)

	args := containerHealthArgs{Container: "nocheck"}
	result, err := handleContainerHealth(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "No healthcheck configured for this container." {
		t.Errorf("expected no healthcheck message, got: %s", result)
	}
}

func TestHandleContainerHealth_ContainerNotFound(t *testing.T) {
	mock := docker.NewMock()

	mock.On("inspect --format {{json .State.Health}} nosuchcontainer", "", fmt.Errorf("Error: No such container: nosuchcontainer"))

	args := containerHealthArgs{Container: "nosuchcontainer"}
	_, err := handleContainerHealth(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "nosuchcontainer") {
		t.Errorf("expected error to mention container name, got: %v", err)
	}
}

func TestHandleContainerHealth_EmptyContainer(t *testing.T) {
	mock := docker.NewMock()

	args := containerHealthArgs{Container: ""}
	_, err := handleContainerHealth(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected an error for empty container, got nil")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected error about required container, got: %v", err)
	}
}

func TestFormatNanoseconds(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{30_000_000_000, "30s"},
		{10_000_000_000, "10s"},
		{60_000_000_000, "1m"},
		{90_000_000_000, "1m30s"},
		{5_000_000_000, "5s"},
	}

	for _, tt := range tests {
		result := formatNanoseconds(tt.input)
		if result != tt.expected {
			t.Errorf("formatNanoseconds(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
