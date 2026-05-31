package cli

import (
	"context"
	"fmt"
	"io"
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
