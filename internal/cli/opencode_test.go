package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestNewOpencodeClient(t *testing.T) {
	client := NewOpencodeClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.BinaryPath != "opencode" {
		t.Errorf("expected BinaryPath to be 'opencode', got %q", client.BinaryPath)
	}
}

func TestNewClient(t *testing.T) {
	t.Run("supported cliName", func(t *testing.T) {
		client, err := NewClient("opencode")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		opClient, ok := client.(*OpencodeClient)
		if !ok {
			t.Fatal("expected OpencodeClient type")
		}
		if opClient.BinaryPath != "opencode" {
			t.Errorf("expected BinaryPath 'opencode', got %q", opClient.BinaryPath)
		}
	})

	t.Run("unsupported cliName", func(t *testing.T) {
		client, err := NewClient("unsupported_cli")
		if err == nil {
			t.Fatal("expected error for unsupported cli, got nil")
		}
		if client != nil {
			t.Errorf("expected nil client for unsupported cli, got %v", client)
		}
	})
}

func TestOpencodeClient_Run(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		agent        string
		prompt       string
		expectedArgs []string
		mockErr      error
	}{
		{
			name:         "all parameters provided",
			model:        "anthropic/claude-3.5",
			agent:        "dev",
			prompt:       "implement login",
			expectedArgs: []string{"-m", "anthropic/claude-3.5", "--agent", "dev", "run", "implement login"},
		},
		{
			name:         "no model, agent provided",
			model:        "",
			agent:        "dev",
			prompt:       "implement login",
			expectedArgs: []string{"--agent", "dev", "run", "implement login"},
		},
		{
			name:         "model provided, no agent",
			model:        "anthropic/claude-3.5",
			agent:        "",
			prompt:       "implement login",
			expectedArgs: []string{"-m", "anthropic/claude-3.5", "run", "implement login"},
		},
		{
			name:         "no model, no agent",
			model:        "",
			agent:        "",
			prompt:       "implement login",
			expectedArgs: []string{"run", "implement login"},
		},
		{
			name:         "command run error propagation",
			model:        "some-model",
			agent:        "some-agent",
			prompt:       "error prompt",
			expectedArgs: []string{"-m", "some-model", "--agent", "some-agent", "run", "error prompt"},
			mockErr:      errors.New("run failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpencodeClient()
			client.BinaryPath = "mock-opencode"

			var calledName string
			var calledArgs []string
			var stdoutBytes, stderrBytes bytes.Buffer

			client.CmdRun = func(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error {
				calledName = name
				calledArgs = args
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

			if calledName != "mock-opencode" {
				t.Errorf("expected BinaryPath %q, got %q", "mock-opencode", calledName)
			}

			if !reflect.DeepEqual(calledArgs, tt.expectedArgs) {
				t.Errorf("expected args %v, got %v", tt.expectedArgs, calledArgs)
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

func TestOpencodeClient_Models(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     string
		mockErr        error
		expectedModels []string
		expectedErr    bool
	}{
		{
			name:           "valid single model",
			mockOutput:     "anthropic/claude-3.5-sonnet",
			expectedModels: []string{"anthropic/claude-3.5-sonnet"},
		},
		{
			name: "multiple models with whitespaces and empty lines",
			mockOutput: `
anthropic/claude-3.5-sonnet
  openai/gpt-4o  
  
google/gemini-pro
`,
			expectedModels: []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-pro"},
		},
		{
			name:        "command output error propagation",
			mockErr:     errors.New("failed to query models"),
			expectedErr: true,
		},
		{
			name:           "empty output",
			mockOutput:     "",
			expectedModels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpencodeClient()
			client.BinaryPath = "mock-opencode"

			var calledName string
			var calledArgs []string

			client.CmdOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				calledName = name
				calledArgs = args
				return []byte(tt.mockOutput), tt.mockErr
			}

			models, err := client.Models(context.Background())

			if tt.expectedErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.mockErr != nil && err.Error() != tt.mockErr.Error() {
					t.Errorf("expected error %q, got %q", tt.mockErr.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(models, tt.expectedModels) {
					t.Errorf("expected models %v, got %v", tt.expectedModels, models)
				}
			}

			if calledName != "mock-opencode" {
				t.Errorf("expected BinaryPath %q, got %q", "mock-opencode", calledName)
			}

			expectedArgs := []string{"models"}
			if !reflect.DeepEqual(calledArgs, expectedArgs) {
				t.Errorf("expected args %v, got %v", expectedArgs, calledArgs)
			}
		})
	}
}
