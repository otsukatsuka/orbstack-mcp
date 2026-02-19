package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleGetLogs_BasicWithDefaults(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "2024-01-01 line1\n2024-01-01 line2\n2024-01-01 line3"
	mock.On("logs --tail 100 mycontainer", logOutput, nil)

	result, err := handleGetLogs(context.Background(), mock, getLogsArgs{
		Container: "mycontainer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != logOutput {
		t.Errorf("expected log output %q, got %q", logOutput, result)
	}
}

func TestHandleGetLogs_WithAllOptions(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "2024-01-01T10:00:00Z line1\n2024-01-01T11:00:00Z line2"
	mock.On("logs --tail 50 --since 1h --until 30m --timestamps mycontainer", logOutput, nil)

	result, err := handleGetLogs(context.Background(), mock, getLogsArgs{
		Container:  "mycontainer",
		Tail:       50,
		Since:      "1h",
		Until:      "30m",
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != logOutput {
		t.Errorf("expected log output %q, got %q", logOutput, result)
	}

	// Verify correct args were passed
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	args := calls[0]
	expectedArgs := []string{"logs", "--tail", "50", "--since", "1h", "--until", "30m", "--timestamps", "mycontainer"}
	if len(args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(args), args)
	}
	for i, a := range args {
		if a != expectedArgs[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expectedArgs[i], a)
		}
	}
}

func TestHandleGetLogs_WithTailAndSince(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "recent log line"
	mock.On("logs --tail 20 --since 2h mycontainer", logOutput, nil)

	result, err := handleGetLogs(context.Background(), mock, getLogsArgs{
		Container: "mycontainer",
		Tail:      20,
		Since:     "2h",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != logOutput {
		t.Errorf("expected %q, got %q", logOutput, result)
	}
}

func TestHandleGetLogs_ContainerNotFound(t *testing.T) {
	mock := docker.NewMock()
	mock.On("logs --tail 100 nonexistent", "", fmt.Errorf("Error: No such container: nonexistent"))

	_, err := handleGetLogs(context.Background(), mock, getLogsArgs{
		Container: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention container name, got: %v", err)
	}
}

func TestHandleGetLogs_EmptyContainer(t *testing.T) {
	_, err := handleGetLogs(context.Background(), docker.NewMock(), getLogsArgs{
		Container: "",
	})
	if err == nil {
		t.Fatal("expected error for empty container name")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' in error message, got: %v", err)
	}
}

func TestHandleGetLogs_EmptyOutput(t *testing.T) {
	mock := docker.NewMock()
	mock.On("logs --tail 100 empty-container", "", nil)

	result, err := handleGetLogs(context.Background(), mock, getLogsArgs{
		Container: "empty-container",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "No log output." {
		t.Errorf("expected 'No log output.', got %q", result)
	}
}
