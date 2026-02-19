package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleListContainers_MixedComposeAndStandalone(t *testing.T) {
	mock := docker.NewMock()

	// Two Compose containers (project "webapp") and one standalone
	lines := strings.Join([]string{
		`{"ID":"abc123","Names":"webapp-web-1","Image":"nginx:latest","State":"running","Status":"Up 2 hours","Ports":"0.0.0.0:80->80/tcp","Labels":"com.docker.compose.project=webapp,com.docker.compose.service=web","CreatedAt":"2024-01-01 00:00:00","Networks":"webapp_default"}`,
		`{"ID":"def456","Names":"webapp-db-1","Image":"postgres:16","State":"running","Status":"Up 2 hours","Ports":"5432/tcp","Labels":"com.docker.compose.project=webapp,com.docker.compose.service=db","CreatedAt":"2024-01-01 00:00:00","Networks":"webapp_default"}`,
		`{"ID":"ghi789","Names":"my-redis","Image":"redis:7","State":"running","Status":"Up 1 hour","Ports":"6379/tcp","Labels":"","CreatedAt":"2024-01-01 01:00:00","Networks":"bridge"}`,
	}, "\n")

	mock.On("ps -a --format {{json .}}", lines, nil)

	result, err := handleListContainers(context.Background(), mock, listContainersArgs{All: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have webapp group before standalone
	if !strings.Contains(result, "=== webapp ===") {
		t.Error("expected webapp group header")
	}
	if !strings.Contains(result, "=== (standalone) ===") {
		t.Error("expected standalone group header")
	}
	if !strings.Contains(result, "webapp-web-1") {
		t.Error("expected webapp-web-1 container")
	}
	if !strings.Contains(result, "webapp-db-1") {
		t.Error("expected webapp-db-1 container")
	}
	if !strings.Contains(result, "my-redis") {
		t.Error("expected my-redis container")
	}

	// webapp group should appear before standalone
	webappIdx := strings.Index(result, "=== webapp ===")
	standaloneIdx := strings.Index(result, "=== (standalone) ===")
	if webappIdx > standaloneIdx {
		t.Error("expected webapp group to appear before standalone group")
	}
}

func TestHandleListContainers_FilterByProject(t *testing.T) {
	mock := docker.NewMock()

	lines := strings.Join([]string{
		`{"ID":"abc123","Names":"webapp-web-1","Image":"nginx:latest","State":"running","Status":"Up 2 hours","Ports":"","Labels":"com.docker.compose.project=webapp","CreatedAt":"","Networks":""}`,
		`{"ID":"def456","Names":"api-server-1","Image":"node:20","State":"running","Status":"Up 1 hour","Ports":"","Labels":"com.docker.compose.project=api","CreatedAt":"","Networks":""}`,
		`{"ID":"ghi789","Names":"my-redis","Image":"redis:7","State":"running","Status":"Up 1 hour","Ports":"","Labels":"","CreatedAt":"","Networks":""}`,
	}, "\n")

	mock.On("ps -a --format {{json .}}", lines, nil)

	result, err := handleListContainers(context.Background(), mock, listContainersArgs{All: true, Project: "webapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "webapp-web-1") {
		t.Error("expected webapp-web-1 container in filtered result")
	}
	if strings.Contains(result, "api-server-1") {
		t.Error("should not contain api-server-1 when filtering by webapp")
	}
	if strings.Contains(result, "my-redis") {
		t.Error("should not contain standalone containers when filtering by project")
	}
	if strings.Contains(result, "(standalone)") {
		t.Error("should not contain standalone group when filtering by project")
	}
}

func TestHandleListContainers_EmptyResult(t *testing.T) {
	mock := docker.NewMock()
	mock.On("ps -a --format {{json .}}", "", nil)

	result, err := handleListContainers(context.Background(), mock, listContainersArgs{All: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "No containers found." {
		t.Errorf("expected 'No containers found.', got %q", result)
	}
}

func TestHandleListContainers_DockerError(t *testing.T) {
	mock := docker.NewMock()
	mock.On("ps -a --format {{json .}}", "", fmt.Errorf("Cannot connect to the Docker daemon"))

	_, err := handleListContainers(context.Background(), mock, listContainersArgs{All: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Errorf("expected docker error message, got: %v", err)
	}
}

func TestHandleListContainers_NotAllFlag(t *testing.T) {
	mock := docker.NewMock()

	line := `{"ID":"abc123","Names":"running-container","Image":"nginx","State":"running","Status":"Up 1 hour","Ports":"","Labels":"","CreatedAt":"","Networks":""}`
	mock.On("ps --format {{json .}}", line, nil)

	result, err := handleListContainers(context.Background(), mock, listContainersArgs{All: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "running-container") {
		t.Error("expected running-container in result")
	}

	// Verify the correct docker command was called (without -a flag)
	calls := mock.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	for _, arg := range calls[0] {
		if arg == "-a" {
			t.Error("should not include -a flag when All is false")
		}
	}
}

func TestHandleListContainers_FilterByProjectNoMatch(t *testing.T) {
	mock := docker.NewMock()

	line := `{"ID":"abc123","Names":"webapp-web-1","Image":"nginx","State":"running","Status":"Up","Ports":"","Labels":"com.docker.compose.project=webapp","CreatedAt":"","Networks":""}`
	mock.On("ps -a --format {{json .}}", line, nil)

	result, err := handleListContainers(context.Background(), mock, listContainersArgs{All: true, Project: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No containers found for project") {
		t.Errorf("expected no containers message for project filter, got %q", result)
	}
}
