package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleContainerStats_SingleContainer(t *testing.T) {
	mock := docker.NewMock()
	mock.On("stats --no-stream --format {{json .}} nginx", `{"Container":"abc123","Name":"nginx","ID":"abc123def456","CPUPerc":"0.50%","MemUsage":"50MiB / 1GiB","MemPerc":"5.00%","NetIO":"1.2kB / 3.4kB","BlockIO":"10MB / 20MB","PIDs":"5"}`, nil)

	args := containerStatsArgs{Container: "nginx"}
	result, err := handleContainerStats(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "nginx") {
		t.Errorf("expected result to contain container name 'nginx', got:\n%s", result)
	}
	if !strings.Contains(result, "0.50%") {
		t.Errorf("expected result to contain CPU percentage '0.50%%', got:\n%s", result)
	}
	if !strings.Contains(result, "50MiB / 1GiB") {
		t.Errorf("expected result to contain memory usage, got:\n%s", result)
	}
	if !strings.Contains(result, "CONTAINER") {
		t.Errorf("expected result to contain table header, got:\n%s", result)
	}
}

func TestHandleContainerStats_AllContainers(t *testing.T) {
	mock := docker.NewMock()
	multiOutput := `{"Container":"abc123","Name":"nginx","ID":"abc123","CPUPerc":"0.50%","MemUsage":"50MiB / 1GiB","MemPerc":"5.00%","NetIO":"1.2kB / 3.4kB","BlockIO":"10MB / 20MB","PIDs":"5"}
{"Container":"def456","Name":"redis","ID":"def456","CPUPerc":"1.20%","MemUsage":"100MiB / 1GiB","MemPerc":"10.00%","NetIO":"5.6kB / 7.8kB","BlockIO":"30MB / 40MB","PIDs":"10"}`
	mock.On("stats --no-stream --format {{json .}}", multiOutput, nil)

	args := containerStatsArgs{Container: ""}
	result, err := handleContainerStats(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "nginx") {
		t.Errorf("expected result to contain 'nginx', got:\n%s", result)
	}
	if !strings.Contains(result, "redis") {
		t.Errorf("expected result to contain 'redis', got:\n%s", result)
	}
	if !strings.Contains(result, "1.20%") {
		t.Errorf("expected result to contain redis CPU '1.20%%', got:\n%s", result)
	}
}

func TestHandleContainerStats_NoRunningContainers(t *testing.T) {
	mock := docker.NewMock()
	mock.On("stats --no-stream --format {{json .}}", "", nil)

	args := containerStatsArgs{Container: ""}
	result, err := handleContainerStats(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "No running containers found." {
		t.Errorf("expected 'No running containers found.', got: %s", result)
	}
}

func TestHandleContainerStats_DockerError(t *testing.T) {
	mock := docker.NewMock()
	mock.On("stats --no-stream --format {{json .}} badcontainer", "", fmt.Errorf("Error: No such container: badcontainer"))

	args := containerStatsArgs{Container: "badcontainer"}
	_, err := handleContainerStats(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "badcontainer") {
		t.Errorf("expected error to mention container name, got: %v", err)
	}
}
