package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/otsukatsuka/orbstack-mcp/docker"
)

func TestHandleSearchLogs_BasicMatch(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "INFO starting server\nERROR connection refused\nINFO request handled\nERROR timeout\nINFO shutting down"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container: "myapp",
		Pattern:   "ERROR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Found 2 matches") {
		t.Errorf("expected 2 matches header, got: %s", result)
	}
	if !strings.Contains(result, "ERROR connection refused") {
		t.Error("expected 'ERROR connection refused' in result")
	}
	if !strings.Contains(result, "ERROR timeout") {
		t.Error("expected 'ERROR timeout' in result")
	}
	if strings.Contains(result, "INFO starting server") {
		t.Error("should not contain non-matching lines")
	}
}

func TestHandleSearchLogs_WithContextLines(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "line1\nline2\nERROR something broke\nline4\nline5\nline6\nline7\nERROR another failure\nline9\nline10"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container:    "myapp",
		Pattern:      "ERROR",
		ContextLines: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Found 2 matches") {
		t.Errorf("expected 2 matches header, got: %s", result)
	}

	// Check context around first match (line2, ERROR something broke, line4)
	if !strings.Contains(result, "  line2") {
		t.Error("expected context line 'line2' before first match")
	}
	if !strings.Contains(result, "> ERROR something broke") {
		t.Error("expected matched line with '>' prefix")
	}
	if !strings.Contains(result, "  line4") {
		t.Error("expected context line 'line4' after first match")
	}

	// Check context around second match (line7, ERROR another failure, line9)
	if !strings.Contains(result, "  line7") {
		t.Error("expected context line 'line7' before second match")
	}
	if !strings.Contains(result, "> ERROR another failure") {
		t.Error("expected second matched line with '>' prefix")
	}
	if !strings.Contains(result, "  line9") {
		t.Error("expected context line 'line9' after second match")
	}

	// The two groups should be separated by "--"
	if !strings.Contains(result, "--") {
		t.Error("expected '--' separator between context groups")
	}
}

func TestHandleSearchLogs_OverlappingContext(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "line1\nERROR first\nline3\nERROR second\nline5"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container:    "myapp",
		Pattern:      "ERROR",
		ContextLines: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With context_lines=2, the ranges around the two ERROR lines overlap
	// so they should be merged into one group without "--" separator
	lines := strings.Split(result, "\n")
	separatorCount := 0
	for _, line := range lines {
		if line == "--" {
			separatorCount++
		}
	}
	if separatorCount != 0 {
		t.Errorf("expected no '--' separator (overlapping context), got %d separators", separatorCount)
	}
}

func TestHandleSearchLogs_InvalidRegex(t *testing.T) {
	mock := docker.NewMock()
	// Don't need to set up mock since it should fail at regex compilation
	_ = mock

	_, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container: "myapp",
		Pattern:   "[invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}

	if !strings.Contains(err.Error(), "invalid regex pattern") {
		t.Errorf("expected regex error message, got: %v", err)
	}
}

func TestHandleSearchLogs_NoMatches(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "INFO all is well\nINFO nothing to see here\nINFO carry on"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container: "myapp",
		Pattern:   "ERROR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected 'No matches found' message, got: %s", result)
	}
	if !strings.Contains(result, "3 log lines") {
		t.Errorf("expected line count in no-match message, got: %s", result)
	}
}

func TestHandleSearchLogs_DockerError(t *testing.T) {
	mock := docker.NewMock()
	mock.On("logs --tail 1000 nonexistent", "", fmt.Errorf("Error: No such container: nonexistent"))

	_, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container: "nonexistent",
		Pattern:   "ERROR",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected container name in error, got: %v", err)
	}
}

func TestHandleSearchLogs_RegexPattern(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "2024-01-01 10:00:00 GET /api/users 200\n2024-01-01 10:00:01 POST /api/users 201\n2024-01-01 10:00:02 GET /api/health 200\n2024-01-01 10:00:03 GET /api/users/123 404"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container: "myapp",
		Pattern:   `(4\d{2}|5\d{2})$`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Found 1 match") {
		t.Errorf("expected 1 match, got: %s", result)
	}
	if !strings.Contains(result, "404") {
		t.Error("expected line with 404")
	}
}

func TestHandleSearchLogs_WithTimestampsAndSince(t *testing.T) {
	mock := docker.NewMock()

	logOutput := "2024-01-01T10:00:00Z ERROR something broke"
	mock.On("logs --tail 500 --since 1h --timestamps myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container:  "myapp",
		Pattern:    "ERROR",
		Tail:       500,
		Since:      "1h",
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "ERROR something broke") {
		t.Errorf("expected match in result, got: %s", result)
	}
}

func TestHandleSearchLogs_EmptyContainer(t *testing.T) {
	_, err := handleSearchLogs(context.Background(), docker.NewMock(), searchLogsArgs{
		Container: "",
		Pattern:   "ERROR",
	})
	if err == nil {
		t.Fatal("expected error for empty container name")
	}
}

func TestHandleSearchLogs_EmptyPattern(t *testing.T) {
	_, err := handleSearchLogs(context.Background(), docker.NewMock(), searchLogsArgs{
		Container: "myapp",
		Pattern:   "",
	})
	if err == nil {
		t.Fatal("expected error for empty pattern")
	}
}

func TestHandleSearchLogs_ContextAtBoundary(t *testing.T) {
	mock := docker.NewMock()

	// Match at the very first and last line
	logOutput := "ERROR first line\nline2\nline3\nline4\nERROR last line"
	mock.On("logs --tail 1000 myapp", logOutput, nil)

	result, err := handleSearchLogs(context.Background(), mock, searchLogsArgs{
		Container:    "myapp",
		Pattern:      "ERROR",
		ContextLines: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic or error with context lines at boundaries
	if !strings.Contains(result, "Found 2 matches") {
		t.Errorf("expected 2 matches, got: %s", result)
	}
	if !strings.Contains(result, "> ERROR first line") {
		t.Error("expected first ERROR line with '>' prefix")
	}
	if !strings.Contains(result, "> ERROR last line") {
		t.Error("expected last ERROR line with '>' prefix")
	}
}
