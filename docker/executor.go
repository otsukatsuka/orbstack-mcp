package docker

import "context"

// Executor abstracts docker CLI execution for testability.
type Executor interface {
	// Exec runs "docker <args>" and returns stdout.
	// Returns error (wrapping stderr) on non-zero exit.
	Exec(ctx context.Context, args ...string) (string, error)

	// ExecCombined runs "docker <args>" and returns combined stdout+stderr.
	// Useful for "docker logs" which outputs to both streams.
	ExecCombined(ctx context.Context, args ...string) (string, error)
}
