package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleContainerExec_BasicCommand(t *testing.T) {
	mock := docker.NewMock()

	mock.On("exec mycontainer sh -c ls -la", "total 0\ndrwxr-xr-x 2 root root 40 Jan  1 00:00 .\n", nil)

	result, err := handleContainerExec(context.Background(), mock, containerExecArgs{
		Container: "mycontainer",
		Command:   "ls -la",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "total 0") {
		t.Errorf("expected command output, got: %s", result)
	}
}

func TestHandleContainerExec_WithUserAndWorkdir(t *testing.T) {
	mock := docker.NewMock()

	mock.On("exec --user www-data --workdir /var/www mycontainer sh -c cat index.html", "<html>hello</html>\n", nil)

	result, err := handleContainerExec(context.Background(), mock, containerExecArgs{
		Container: "mycontainer",
		Command:   "cat index.html",
		User:      "www-data",
		Workdir:   "/var/www",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "<html>hello</html>") {
		t.Errorf("expected html output, got: %s", result)
	}
}

func TestHandleContainerExec_CommandFails(t *testing.T) {
	mock := docker.NewMock()

	mock.On("exec mycontainer sh -c exit 1", "", fmt.Errorf("exit status 1"))

	_, err := handleContainerExec(context.Background(), mock, containerExecArgs{
		Container: "mycontainer",
		Command:   "exit 1",
	})

	if err == nil {
		t.Fatal("expected error for failed command, got nil")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Errorf("expected 'exec failed' error, got: %v", err)
	}
}

func TestHandleContainerExec_ContainerNotFound(t *testing.T) {
	mock := docker.NewMock()

	mock.On("exec nosuchcontainer sh -c echo hello", "", fmt.Errorf("Error: No such container: nosuchcontainer"))

	_, err := handleContainerExec(context.Background(), mock, containerExecArgs{
		Container: "nosuchcontainer",
		Command:   "echo hello",
	})

	if err == nil {
		t.Fatal("expected error for missing container, got nil")
	}
	if !strings.Contains(err.Error(), "exec failed") {
		t.Errorf("expected 'exec failed' error, got: %v", err)
	}
}
