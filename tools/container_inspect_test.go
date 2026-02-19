package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

const inspectJSON = `[{
  "Id": "abc123",
  "Config": {
    "Env": [
      "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
      "NGINX_VERSION=1.25.0",
      "MY_VAR=hello"
    ]
  },
  "HostConfig": {
    "PortBindings": {
      "80/tcp": [
        {"HostIp": "", "HostPort": "8080"}
      ],
      "443/tcp": [
        {"HostIp": "127.0.0.1", "HostPort": "8443"}
      ]
    }
  },
  "Mounts": [
    {
      "Type": "bind",
      "Source": "/host/data",
      "Destination": "/container/data",
      "RW": true
    },
    {
      "Type": "volume",
      "Source": "my-volume",
      "Destination": "/container/vol",
      "RW": false
    }
  ],
  "NetworkSettings": {
    "Ports": {
      "80/tcp": [
        {"HostIp": "0.0.0.0", "HostPort": "8080"}
      ],
      "443/tcp": [
        {"HostIp": "127.0.0.1", "HostPort": "8443"}
      ]
    },
    "Networks": {
      "bridge": {
        "IPAddress": "172.17.0.2",
        "Gateway": "172.17.0.1",
        "MacAddress": "02:42:ac:11:00:02"
      }
    }
  }
}]`

func TestHandleContainerInspect_AllSection(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect mycontainer", inspectJSON, nil)

	args := containerInspectArgs{Container: "mycontainer", Section: "all"}
	result, err := handleContainerInspect(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain pretty-printed JSON
	if !strings.Contains(result, "abc123") {
		t.Errorf("expected full inspect output to contain container ID, got:\n%s", result)
	}
	if !strings.Contains(result, "NGINX_VERSION") {
		t.Errorf("expected full inspect output to contain env vars, got:\n%s", result)
	}
}

func TestHandleContainerInspect_EnvSection(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect mycontainer", inspectJSON, nil)

	args := containerInspectArgs{Container: "mycontainer", Section: "env"}
	result, err := handleContainerInspect(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Environment Variables:") {
		t.Errorf("expected 'Environment Variables:' header, got:\n%s", result)
	}
	if !strings.Contains(result, "NGINX_VERSION=1.25.0") {
		t.Errorf("expected env var NGINX_VERSION, got:\n%s", result)
	}
	if !strings.Contains(result, "MY_VAR=hello") {
		t.Errorf("expected env var MY_VAR, got:\n%s", result)
	}
}

func TestHandleContainerInspect_PortsSection(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect mycontainer", inspectJSON, nil)

	args := containerInspectArgs{Container: "mycontainer", Section: "ports"}
	result, err := handleContainerInspect(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Port Bindings:") {
		t.Errorf("expected 'Port Bindings:' header, got:\n%s", result)
	}
	if !strings.Contains(result, "80/tcp") {
		t.Errorf("expected port 80/tcp, got:\n%s", result)
	}
	if !strings.Contains(result, "8080") {
		t.Errorf("expected host port 8080, got:\n%s", result)
	}
}

func TestHandleContainerInspect_VolumesSection(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect mycontainer", inspectJSON, nil)

	args := containerInspectArgs{Container: "mycontainer", Section: "volumes"}
	result, err := handleContainerInspect(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Mounts:") {
		t.Errorf("expected 'Mounts:' header, got:\n%s", result)
	}
	if !strings.Contains(result, "/host/data") {
		t.Errorf("expected source path, got:\n%s", result)
	}
	if !strings.Contains(result, "/container/data") {
		t.Errorf("expected destination path, got:\n%s", result)
	}
	if !strings.Contains(result, "[bind]") {
		t.Errorf("expected mount type 'bind', got:\n%s", result)
	}
	if !strings.Contains(result, "(ro)") {
		t.Errorf("expected read-only mount indicator, got:\n%s", result)
	}
}

func TestHandleContainerInspect_DefaultSection(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect mycontainer", inspectJSON, nil)

	// Empty section should default to "all"
	args := containerInspectArgs{Container: "mycontainer", Section: ""}
	result, err := handleContainerInspect(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "abc123") {
		t.Errorf("expected default section (all) to contain container ID, got:\n%s", result)
	}
}

func TestHandleContainerInspect_ContainerNotFound(t *testing.T) {
	mock := docker.NewMock()
	mock.On("inspect nosuchcontainer", "", fmt.Errorf("Error: No such container: nosuchcontainer"))

	args := containerInspectArgs{Container: "nosuchcontainer", Section: "all"}
	_, err := handleContainerInspect(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "nosuchcontainer") {
		t.Errorf("expected error to mention container name, got: %v", err)
	}
}

func TestHandleContainerInspect_EmptyContainer(t *testing.T) {
	mock := docker.NewMock()

	args := containerInspectArgs{Container: "", Section: "all"}
	_, err := handleContainerInspect(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected an error for empty container name, got nil")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected error about required container, got: %v", err)
	}
}
