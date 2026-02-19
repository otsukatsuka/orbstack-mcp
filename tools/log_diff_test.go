package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleLogDiff_BasicDiff(t *testing.T) {
	mock := docker.NewMock()

	period1Logs := "INFO: server started\nERROR: connection timeout\nINFO: request handled"
	period2Logs := "INFO: server started\nERROR: disk full\nINFO: request handled"

	mock.On("logs --since 2h --until 1h myapp", period1Logs, nil)
	mock.On("logs --since 1h --until now myapp", period2Logs, nil)

	args := logDiffArgs{
		Container:    "myapp",
		Period1Start: "2h",
		Period1End:   "1h",
		Period2Start: "1h",
		Period2End:   "now",
	}

	result, err := handleLogDiff(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should show "connection timeout" only in period 1
	if !strings.Contains(result, "ERROR: connection timeout") {
		t.Error("expected 'ERROR: connection timeout' in Only in Period 1 section")
	}

	// Should show "disk full" only in period 2
	if !strings.Contains(result, "ERROR: disk full") {
		t.Error("expected 'ERROR: disk full' in Only in Period 2 section")
	}

	// Common lines should be present
	if !strings.Contains(result, "INFO: server started") {
		t.Error("expected 'INFO: server started' in common section")
	}
	if !strings.Contains(result, "INFO: request handled") {
		t.Error("expected 'INFO: request handled' in common section")
	}

	// Verify section headers
	if !strings.Contains(result, "--- Only in Period 1 ---") {
		t.Error("expected 'Only in Period 1' section header")
	}
	if !strings.Contains(result, "--- Only in Period 2 ---") {
		t.Error("expected 'Only in Period 2' section header")
	}
}

func TestHandleLogDiff_IdenticalLogs(t *testing.T) {
	mock := docker.NewMock()

	logs := "INFO: all good\nINFO: running"
	mock.On("logs --since 2h --until 1h myapp", logs, nil)
	mock.On("logs --since 1h --until now myapp", logs, nil)

	args := logDiffArgs{
		Container:    "myapp",
		Period1Start: "2h",
		Period1End:   "1h",
		Period2Start: "1h",
		Period2End:   "now",
	}

	result, err := handleLogDiff(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both "only in" sections should show (none)
	sections := strings.Split(result, "---")
	for _, section := range sections {
		if strings.Contains(section, "Only in Period 1") || strings.Contains(section, "Only in Period 2") {
			// The next section after the header should have (none)
		}
	}

	// Check that the unique sections contain (none)
	// Find the "Only in Period 1" section
	p1Idx := strings.Index(result, "--- Only in Period 1 ---")
	p2Idx := strings.Index(result, "--- Only in Period 2 ---")
	if p1Idx < 0 || p2Idx < 0 {
		t.Fatal("missing section headers")
	}
	p1Section := result[p1Idx:p2Idx]
	if !strings.Contains(p1Section, "(none)") {
		t.Error("expected (none) in Only in Period 1 section for identical logs")
	}

	countIdx := strings.Index(result, "--- Count Changes ---")
	if countIdx < 0 {
		t.Fatal("missing Count Changes section header")
	}
	p2Section := result[p2Idx:countIdx]
	if !strings.Contains(p2Section, "(none)") {
		t.Error("expected (none) in Only in Period 2 section for identical logs")
	}

	// Common section should have both lines
	if !strings.Contains(result, "INFO: all good") {
		t.Error("expected 'INFO: all good' in common section")
	}
	if !strings.Contains(result, "INFO: running") {
		t.Error("expected 'INFO: running' in common section")
	}
}

func TestHandleLogDiff_EmptyPeriod(t *testing.T) {
	mock := docker.NewMock()

	mock.On("logs --since 2h --until 1h myapp", "ERROR: something broke\nWARN: disk usage high", nil)
	mock.On("logs --since 1h --until now myapp", "", nil)

	args := logDiffArgs{
		Container:    "myapp",
		Period1Start: "2h",
		Period1End:   "1h",
		Period2Start: "1h",
		Period2End:   "now",
	}

	result, err := handleLogDiff(context.Background(), mock, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Period 1 lines should be "only in period 1"
	if !strings.Contains(result, "ERROR: something broke") {
		t.Error("expected 'ERROR: something broke' in Only in Period 1")
	}

	// Period 2 unique section should be (none)
	p2Idx := strings.Index(result, "--- Only in Period 2 ---")
	countIdx := strings.Index(result, "--- Count Changes ---")
	if p2Idx < 0 || countIdx < 0 {
		t.Fatal("missing section headers")
	}
	p2Section := result[p2Idx:countIdx]
	if !strings.Contains(p2Section, "(none)") {
		t.Error("expected (none) in Only in Period 2 section")
	}
}

func TestHandleLogDiff_DockerError(t *testing.T) {
	mock := docker.NewMock()

	mock.On("logs --since 2h --until 1h myapp", "", fmt.Errorf("container not found"))

	args := logDiffArgs{
		Container:    "myapp",
		Period1Start: "2h",
		Period1End:   "1h",
		Period2Start: "1h",
		Period2End:   "now",
	}

	_, err := handleLogDiff(context.Background(), mock, args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "period 1") {
		t.Errorf("expected error mentioning 'period 1', got: %v", err)
	}
}
