package cli

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

// OpencodeClient implements the Client interface for the opencode CLI tool.
type OpencodeClient struct {
	// BinaryPath is the path or name of the opencode executable. Defaults to "opencode".
	BinaryPath string

	// CmdRun is an optional override for running commands. It is primarily used for mocking in unit tests.
	CmdRun func(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error

	// CmdOutput is an optional override for running commands and capturing stdout. It is primarily used for mocking in unit tests.
	CmdOutput func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewOpencodeClient creates a new OpencodeClient driver with default values.
func NewOpencodeClient() *OpencodeClient {
	return &OpencodeClient{
		BinaryPath: "opencode",
	}
}

// Run executes an opencode CLI session with the given model, agent, and prompt.
// It streams stdout and stderr to the specified writers.
func (c *OpencodeClient) Run(ctx context.Context, model string, agent string, prompt string, stdout, stderr io.Writer) error {
	args := []string{}
	if model != "" {
		args = append(args, "-m", model)
	}
	if agent != "" {
		args = append(args, "--agent", agent)
	}
	args = append(args, "run", prompt)

	if c.CmdRun != nil {
		return c.CmdRun(ctx, stdout, stderr, c.BinaryPath, args...)
	}

	cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// Models queries the opencode CLI for the list of supported LLM models.
func (c *OpencodeClient) Models(ctx context.Context) ([]string, error) {
	args := []string{"models"}

	var output []byte
	var err error
	if c.CmdOutput != nil {
		output, err = c.CmdOutput(ctx, c.BinaryPath, args...)
	} else {
		cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
		output, err = cmd.Output()
	}

	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var models []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			models = append(models, trimmed)
		}
	}
	return models, nil
}
