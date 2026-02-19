package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleContainerEvents_Basic(t *testing.T) {
	mock := docker.NewMock()

	eventsOutput := `{"status":"start","Action":"start","Type":"container","Actor":{"ID":"abc123def456","Attributes":{"name":"myapp","image":"nginx:latest"}},"time":1700000000}
{"status":"die","Action":"die","Type":"container","Actor":{"ID":"abc123def456","Attributes":{"name":"myapp","image":"nginx:latest","exitCode":"0"}},"time":1700003600}`

	mock.On("events --filter type=container --since 1h --until now --format {{json .}}", eventsOutput, nil)

	args := containerEventsArgs{}

	result, err := handleContainerEvents(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "myapp") {
		t.Error("expected result to contain container name 'myapp'")
	}
	if !strings.Contains(result, "start") {
		t.Error("expected result to contain 'start' event")
	}
	if !strings.Contains(result, "die") {
		t.Error("expected result to contain 'die' event")
	}
	if !strings.Contains(result, "=== Container Events ===") {
		t.Error("expected result to contain header")
	}
}

func TestHandleContainerEvents_FilterByContainer(t *testing.T) {
	mock := docker.NewMock()

	eventsOutput := `{"status":"start","Action":"start","Type":"container","Actor":{"ID":"abc123","Attributes":{"name":"webapp"}},"time":1700000000}`

	mock.On("events --filter type=container --filter container=webapp --since 1h --until now --format {{json .}}", eventsOutput, nil)

	args := containerEventsArgs{
		Container: "webapp",
	}

	result, err := handleContainerEvents(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "webapp") {
		t.Error("expected result to contain 'webapp'")
	}
	if !strings.Contains(result, "Container: webapp") {
		t.Error("expected result to show container filter")
	}
}

func TestHandleContainerEvents_FilterByEventType(t *testing.T) {
	mock := docker.NewMock()

	eventsOutput := `{"status":"oom","Action":"oom","Type":"container","Actor":{"ID":"abc123","Attributes":{"name":"memhog"}},"time":1700000000}`

	mock.On("events --filter type=container --filter event=oom --since 24h --until now --format {{json .}}", eventsOutput, nil)

	args := containerEventsArgs{
		EventType: "oom",
		Since:     "24h",
	}

	result, err := handleContainerEvents(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "oom") {
		t.Error("expected result to contain 'oom' event")
	}
	if !strings.Contains(result, "Event filter: oom") {
		t.Error("expected result to show event type filter")
	}
}

func TestHandleContainerEvents_NoEvents(t *testing.T) {
	mock := docker.NewMock()

	mock.On("events --filter type=container --since 1h --until now --format {{json .}}", "", nil)

	args := containerEventsArgs{}

	result, err := handleContainerEvents(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No events found") {
		t.Errorf("expected 'No events found' message, got: %s", result)
	}
}

func TestHandleContainerEvents_DockerError(t *testing.T) {
	mock := docker.NewMock()

	mock.On("events --filter type=container --since 1h --until now --format {{json .}}", "", fmt.Errorf("daemon connection refused"))

	args := containerEventsArgs{}

	_, err := handleContainerEvents(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "daemon connection refused") {
		t.Errorf("expected daemon connection error, got: %v", err)
	}
}

func TestHandleContainerEvents_CustomTimeRange(t *testing.T) {
	mock := docker.NewMock()

	eventsOutput := `{"status":"restart","Action":"restart","Type":"container","Actor":{"ID":"def456","Attributes":{"name":"api-server"}},"time":1700050000}`

	mock.On("events --filter type=container --since 2024-01-01T00:00:00Z --until 2024-01-02T00:00:00Z --format {{json .}}", eventsOutput, nil)

	args := containerEventsArgs{
		Since: "2024-01-01T00:00:00Z",
		Until: "2024-01-02T00:00:00Z",
	}

	result, err := handleContainerEvents(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "api-server") {
		t.Error("expected result to contain 'api-server'")
	}
	if !strings.Contains(result, "restart") {
		t.Error("expected result to contain 'restart' event")
	}
	if !strings.Contains(result, "2024-01-01T00:00:00Z") {
		t.Error("expected result to show custom time range")
	}
}
