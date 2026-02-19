package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleComposeUp_ExistingProject(t *testing.T) {
	mock := docker.NewMock()

	psOutput := `{"ID":"abc123","Names":"myproject-web-1","State":"running","Status":"Up 2 hours","Labels":"com.docker.compose.project=myproject"}`
	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=myproject`, psOutput, nil)
	mock.On(`inspect --format {{index .Config.Labels "com.docker.compose.project.working_dir"}} abc123`, "/home/user/myproject\n", nil)
	mock.On("compose --project-directory /home/user/myproject -p myproject up -d", "Creating myproject-web-1 ... done\n", nil)

	args := composeUpArgs{
		Project: "myproject",
	}

	result, err := handleComposeUp(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "myproject") {
		t.Error("expected result to mention project name")
	}
	if !strings.Contains(result, "/home/user/myproject") {
		t.Error("expected result to mention working directory")
	}
	if !strings.Contains(result, "started") {
		t.Error("expected result to mention 'started'")
	}
}

func TestHandleComposeUp_WithServices(t *testing.T) {
	mock := docker.NewMock()

	psOutput := `{"ID":"abc123","Names":"myproject-web-1","State":"running"}`
	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=myproject`, psOutput, nil)
	mock.On(`inspect --format {{index .Config.Labels "com.docker.compose.project.working_dir"}} abc123`, "/home/user/myproject\n", nil)
	mock.On("compose --project-directory /home/user/myproject -p myproject up -d web redis", "Creating myproject-web-1 ... done\nCreating myproject-redis-1 ... done\n", nil)

	args := composeUpArgs{
		Project:  "myproject",
		Services: []string{"web", "redis"},
	}

	result, err := handleComposeUp(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "started") {
		t.Error("expected result to indicate success")
	}
}

func TestHandleComposeDown_ExistingProject(t *testing.T) {
	mock := docker.NewMock()

	psOutput := `{"ID":"abc123","Names":"myproject-web-1","State":"running"}`
	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=myproject`, psOutput, nil)
	mock.On(`inspect --format {{index .Config.Labels "com.docker.compose.project.working_dir"}} abc123`, "/home/user/myproject\n", nil)
	mock.On("compose --project-directory /home/user/myproject -p myproject down", "Stopping myproject-web-1 ... done\nRemoving myproject-web-1 ... done\n", nil)

	args := composeDownArgs{
		Project: "myproject",
	}

	result, err := handleComposeDown(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "stopped") {
		t.Error("expected result to mention 'stopped'")
	}
	if !strings.Contains(result, "/home/user/myproject") {
		t.Error("expected result to mention working directory")
	}
}

func TestHandleComposeDown_RemoveVolumes(t *testing.T) {
	mock := docker.NewMock()

	psOutput := `{"ID":"abc123","Names":"myproject-db-1","State":"exited"}`
	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=myproject`, psOutput, nil)
	mock.On(`inspect --format {{index .Config.Labels "com.docker.compose.project.working_dir"}} abc123`, "/home/user/myproject\n", nil)
	mock.On("compose --project-directory /home/user/myproject -p myproject down --volumes", "Removing volume myproject_db-data ... done\n", nil)

	args := composeDownArgs{
		Project:       "myproject",
		RemoveVolumes: true,
	}

	result, err := handleComposeDown(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "stopped") {
		t.Error("expected result to mention 'stopped'")
	}
}

func TestHandleComposeUp_ProjectNotFound(t *testing.T) {
	mock := docker.NewMock()

	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=ghost`, "", nil)

	args := composeUpArgs{
		Project: "ghost",
	}

	_, err := handleComposeUp(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
	if !strings.Contains(err.Error(), "no containers found") {
		t.Errorf("expected 'no containers found' error, got: %v", err)
	}
}

func TestHandleComposeDown_ProjectNotFound(t *testing.T) {
	mock := docker.NewMock()

	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=ghost`, "", nil)

	args := composeDownArgs{
		Project: "ghost",
	}

	_, err := handleComposeDown(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
	if !strings.Contains(err.Error(), "no containers found") {
		t.Errorf("expected 'no containers found' error, got: %v", err)
	}
}

func TestHandleComposeUp_DockerError(t *testing.T) {
	mock := docker.NewMock()

	mock.On(`ps -a --format {{json .}} --filter label=com.docker.compose.project=myproject`, "", fmt.Errorf("docker daemon not running"))

	args := composeUpArgs{
		Project: "myproject",
	}

	_, err := handleComposeUp(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "docker daemon") {
		t.Errorf("expected docker daemon error, got: %v", err)
	}
}
