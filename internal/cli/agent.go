package cli

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

// AgentClient implements the Client interface for the agent CLI tool.
type AgentClient struct {
	// BinaryPath is the path or name of the agent executable. Defaults to "agent".
	BinaryPath string

	// CmdRun is an optional override for running commands. It is primarily used for mocking in unit tests.
	CmdRun func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error
}

// NewAgentClient creates a new AgentClient driver with default values.
func NewAgentClient() *AgentClient {
	return &AgentClient{
		BinaryPath: "agent",
	}
}

// Run executes an agent CLI session with the given model (resolved to adapter name) and prompt.
// It feeds the prompt into STDIN of the subprocess and streams stdout and stderr to the specified writers.
func (c *AgentClient) Run(ctx context.Context, model string, agent string, prompt string, stdout, stderr io.Writer) error {
	adapterName := strings.Split(model, "/")[0]
	args := []string{}
	if adapterName != "" {
		args = append(args, adapterName)
	}

	if c.CmdRun != nil {
		return c.CmdRun(ctx, strings.NewReader(prompt), stdout, stderr, c.BinaryPath, args...)
	}

	cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
