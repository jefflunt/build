package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestNewAgentClient(t *testing.T) {
	client := NewAgentClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.BinaryPath != "agent" {
		t.Errorf("expected BinaryPath to be 'agent', got %q", client.BinaryPath)
	}
}

func TestAgentClient_Run(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		agent        string
		prompt       string
		expectedArgs []string
		mockErr      error
	}{
		{
			name:         "model provided as provider/model",
			model:        "test_opencode/some_model",
			agent:        "dev",
			prompt:       "implement login",
			expectedArgs: []string{"test_opencode"},
		},
		{
			name:         "model provided with trailing slash",
			model:        "test_opencode/",
			agent:        "dev",
			prompt:       "implement login",
			expectedArgs: []string{"test_opencode"},
		},
		{
			name:         "command run error propagation",
			model:        "test_claude",
			agent:        "dev",
			prompt:       "error prompt",
			expectedArgs: []string{"test_claude"},
			mockErr:      errors.New("run failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAgentClient()
			client.BinaryPath = "mock-agent"

			var calledName string
			var calledArgs []string
			var calledStdin string
			var stdoutBytes, stderrBytes bytes.Buffer

			client.CmdRun = func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
				calledName = name
				calledArgs = args
				stdinBytes, _ := io.ReadAll(stdin)
				calledStdin = string(stdinBytes)
				_, _ = stdout.Write([]byte("mock stdout"))
				_, _ = stderr.Write([]byte("mock stderr"))
				return tt.mockErr
			}

			err := client.Run(context.Background(), tt.model, tt.agent, tt.prompt, &stdoutBytes, &stderrBytes)

			if tt.mockErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.mockErr.Error() {
					t.Errorf("expected error %q, got %q", tt.mockErr.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if calledName != "mock-agent" {
				t.Errorf("expected BinaryPath %q, got %q", "mock-agent", calledName)
			}

			if !reflect.DeepEqual(calledArgs, tt.expectedArgs) {
				t.Errorf("expected args %v, got %v", tt.expectedArgs, calledArgs)
			}

			if calledStdin != tt.prompt {
				t.Errorf("expected stdin to contain prompt %q, got %q", tt.prompt, calledStdin)
			}

			if stdoutBytes.String() != "mock stdout" {
				t.Errorf("expected stdout %q, got %q", "mock stdout", stdoutBytes.String())
			}

			if stderrBytes.String() != "mock stderr" {
				t.Errorf("expected stderr %q, got %q", "mock stderr", stderrBytes.String())
			}
		})
	}
}
