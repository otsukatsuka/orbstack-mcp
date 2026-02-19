package docker

import (
	"context"
	"fmt"
	"strings"
)

// Mock implements Executor for testing.
// Register expected command outputs with On().
type Mock struct {
	calls   [][]string
	results map[string]mockResult
}

type mockResult struct {
	output string
	err    error
}

func NewMock() *Mock {
	return &Mock{
		results: make(map[string]mockResult),
	}
}

// On registers a response for a specific docker command.
// The key is the joined args (e.g., "ps --format {{json .}}").
func (m *Mock) On(args string, output string, err error) {
	m.results[args] = mockResult{output: output, err: err}
}

// Calls returns all recorded invocations.
func (m *Mock) Calls() [][]string {
	return m.calls
}

func (m *Mock) Exec(ctx context.Context, args ...string) (string, error) {
	return m.exec(args)
}

func (m *Mock) ExecCombined(ctx context.Context, args ...string) (string, error) {
	return m.exec(args)
}

func (m *Mock) exec(args []string) (string, error) {
	m.calls = append(m.calls, args)
	key := strings.Join(args, " ")
	if r, ok := m.results[key]; ok {
		return r.output, r.err
	}
	return "", fmt.Errorf("unexpected docker command: docker %s", key)
}
