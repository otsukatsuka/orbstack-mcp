package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleRestartService_BasicRestart(t *testing.T) {
	mock := docker.NewMock()

	mock.On("restart --time 10 mycontainer", "mycontainer\n", nil)

	result, err := handleRestartService(context.Background(), mock, restartServiceArgs{
		Container: "mycontainer",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Successfully restarted container mycontainer") {
		t.Errorf("expected success message, got: %s", result)
	}
}

func TestHandleRestartService_WithCustomTimeout(t *testing.T) {
	mock := docker.NewMock()

	mock.On("restart --time 30 mycontainer", "mycontainer\n", nil)

	result, err := handleRestartService(context.Background(), mock, restartServiceArgs{
		Container: "mycontainer",
		Timeout:   30,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Successfully restarted container mycontainer") {
		t.Errorf("expected success message, got: %s", result)
	}
}

func TestHandleRestartService_ContainerNotFound(t *testing.T) {
	mock := docker.NewMock()

	mock.On("restart --time 10 nosuchcontainer", "", fmt.Errorf("Error: No such container: nosuchcontainer"))

	_, err := handleRestartService(context.Background(), mock, restartServiceArgs{
		Container: "nosuchcontainer",
	})

	if err == nil {
		t.Fatal("expected error for missing container, got nil")
	}
	if !strings.Contains(err.Error(), "restart failed") {
		t.Errorf("expected 'restart failed' error, got: %v", err)
	}
}
