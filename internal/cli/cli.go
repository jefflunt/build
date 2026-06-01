package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Client standardizes how the orchestrator communicates with an LLM CLI tool.
type Client interface {
	// Run executes an LLM CLI tool session using the specified model, agent, and prompt.
	// It streams the command's stdout and stderr to the provided writers.
	Run(ctx context.Context, model string, agent string, prompt string, stdout io.Writer, stderr io.Writer) error

	// Models retrieves the list of supported/available LLM models from the CLI tool.
	Models(ctx context.Context) ([]string, error)
}

// NewClient creates a new Client implementation for the given CLI name.
func NewClient(cliName string) (Client, error) {
	switch cliName {
	case "opencode":
		return NewOpencodeClient(), nil
	default:
		return nil, fmt.Errorf("unsupported agent_adapter CLI: %s", cliName)
	}
}

// ParseRMArgs parses and validates the arguments for the 'rm' subcommand.
// It expects the slice of command-line arguments (including the program name and subcommand).
// It returns the target string and a boolean indicating whether the arguments are valid.
func ParseRMArgs(args []string) (string, bool) {
	if len(args) != 3 {
		return "", false
	}
	target := args[2]
	if !strings.HasPrefix(target, "id:") && !strings.HasPrefix(target, "status:") {
		return "", false
	}
	if target == "id:" || target == "status:" {
		return "", false
	}
	return target, true
}
