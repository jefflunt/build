package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("unexpected error getting user home dir: %v", err)
	}

	expected := filepath.Join(home, ".build", "config.yml")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestStripCommentsAndQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no quotes no comments", "opencode:anthropic/claude-3.5-sonnet", "opencode:anthropic/claude-3.5-sonnet"},
		{"double quotes", "\"opencode:anthropic/claude-3.5-sonnet\"", "opencode:anthropic/claude-3.5-sonnet"},
		{"single quotes", "'opencode:anthropic/claude-3.5-sonnet'", "opencode:anthropic/claude-3.5-sonnet"},
		{"trailing comment", "opencode:anthropic/claude-3.5-sonnet # comment here", "opencode:anthropic/claude-3.5-sonnet"},
		{"double quotes with trailing comment", "\"opencode:anthropic/claude-3.5-sonnet\" # comment here", "opencode:anthropic/claude-3.5-sonnet"},
		{"single quotes with trailing comment", "'opencode:anthropic/claude-3.5-sonnet' # comment here", "opencode:anthropic/claude-3.5-sonnet"},
		{"unmatched double quote prefix", "\"opencode:anthropic/claude-3.5-sonnet", "\"opencode:anthropic/claude-3.5-sonnet"},
		{"unmatched single quote prefix", "'opencode:anthropic/claude-3.5-sonnet", "'opencode:anthropic/claude-3.5-sonnet"},
		{"with inner comments inside quotes", "\"opencode:anthropic/claude-3.5-sonnet #notacomment\"", "opencode:anthropic/claude-3.5-sonnet #notacomment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCommentsAndQuotes(tt.input)
			if got != tt.expected {
				t.Errorf("stripCommentsAndQuotes(%q) = %q; expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseAdapter(t *testing.T) {
	tests := []struct {
		name          string
		adapter       string
		expected      *Config
		expectedError string
	}{
		{
			name:    "valid",
			adapter: "opencode:anthropic/claude-3.5-sonnet",
			expected: &Config{
				AgentAdapter: "opencode:anthropic/claude-3.5-sonnet",
				CLIName:      "opencode",
				Provider:     "anthropic",
				Model:        "claude-3.5-sonnet",
			},
		},
		{
			name:    "valid agent adapter with space",
			adapter: "agent: test_opencode",
			expected: &Config{
				AgentAdapter: "agent: test_opencode",
				CLIName:      "agent",
				Provider:     "test_opencode",
				Model:        "",
			},
		},
		{
			name:    "valid agent adapter no space",
			adapter: "agent:test_opencode",
			expected: &Config{
				AgentAdapter: "agent:test_opencode",
				CLIName:      "agent",
				Provider:     "test_opencode",
				Model:        "",
			},
		},
		{
			name:          "missing colon",
			adapter:       "opencode-anthropic/claude-3.5-sonnet",
			expectedError: "invalid agent_adapter format",
		},
		{
			name:          "empty CLI name",
			adapter:       ":anthropic/claude-3.5-sonnet",
			expectedError: "CLI name cannot be empty",
		},
		{
			name:          "missing slash",
			adapter:       "opencode:anthropic-claude-3.5-sonnet",
			expectedError: "expected format 'provider/model' after CLI name",
		},
		{
			name:          "empty provider",
			adapter:       "opencode:/claude-3.5-sonnet",
			expectedError: "provider cannot be empty",
		},
		{
			name:          "empty model",
			adapter:       "opencode:anthropic/",
			expectedError: "model cannot be empty",
		},
		{
			name:    "with spacing",
			adapter: " opencode : anthropic / claude-3.5-sonnet ",
			// ParseAdapter handles spaces by trimming parts
			expected: &Config{
				AgentAdapter: " opencode : anthropic / claude-3.5-sonnet ",
				CLIName:      "opencode",
				Provider:     "anthropic",
				Model:        "claude-3.5-sonnet",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAdapter(tt.adapter)
			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.AgentAdapter != tt.expected.AgentAdapter ||
					got.CLIName != tt.expected.CLIName ||
					got.Provider != tt.expected.Provider ||
					got.Model != tt.expected.Model {
					t.Errorf("ParseAdapter() = %+v; expected %+v", got, tt.expected)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		expected      *Config
		expectedError string
	}{
		{
			name: "valid minimal",
			yaml: "agent_adapter: opencode:anthropic/claude-3.5-sonnet",
			expected: &Config{
				AgentAdapter: "opencode:anthropic/claude-3.5-sonnet",
				CLIName:      "opencode",
				Provider:     "anthropic",
				Model:        "claude-3.5-sonnet",
			},
		},
		{
			name: "valid with comments and quotes",
			yaml: `
# This is a comment
agent_adapter: "opencode:anthropic/claude-3.5-sonnet" # use this adapter
`,
			expected: &Config{
				AgentAdapter: "opencode:anthropic/claude-3.5-sonnet",
				CLIName:      "opencode",
				Provider:     "anthropic",
				Model:        "claude-3.5-sonnet",
			},
		},
		{
			name:          "missing field",
			yaml:          "some_other_key: value",
			expectedError: "missing or empty 'agent_adapter' field in configuration",
		},
		{
			name:          "empty field",
			yaml:          "agent_adapter: ",
			expectedError: "missing or empty 'agent_adapter' field in configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.yaml))
			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.AgentAdapter != tt.expected.AgentAdapter ||
					got.CLIName != tt.expected.CLIName ||
					got.Provider != tt.expected.Provider ||
					got.Model != tt.expected.Model {
					t.Errorf("Parse() = %+v; expected %+v", got, tt.expected)
				}
			}
		})
	}
}

type mockConfigSource struct {
	data []byte
	err  error
}

func (m *mockConfigSource) Load() ([]byte, error) {
	return m.data, m.err
}

func TestLoadFromSource(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockConfigSource{
			data: []byte("agent_adapter: opencode:anthropic/claude-3.5-sonnet"),
		}
		got, err := LoadFromSource(mock)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.CLIName != "opencode" || got.Provider != "anthropic" || got.Model != "claude-3.5-sonnet" {
			t.Errorf("unexpected parsed config: %+v", got)
		}
	})

	t.Run("load error", func(t *testing.T) {
		mock := &mockConfigSource{
			err: errors.New("failed to read"),
		}
		_, err := LoadFromSource(mock)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read") {
			t.Errorf("expected error containing 'failed to read', got %q", err.Error())
		}
	})
}

func TestFileConfigSource_Load(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config-file-source-test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	expectedContent := []byte("agent_adapter: opencode:anthropic/claude-3.5-sonnet")
	if _, err := tempFile.Write(expectedContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tempFile.Close()

	source := &FileConfigSource{Path: tempFile.Name()}
	data, err := source.Load()
	if err != nil {
		t.Fatalf("unexpected error loading: %v", err)
	}

	if string(data) != string(expectedContent) {
		t.Errorf("expected %q, got %q", string(expectedContent), string(data))
	}
}

func TestLoad(t *testing.T) {
	// Keep track of original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	tempHome, err := os.MkdirTemp("", "home-test")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	os.Setenv("HOME", tempHome)

	// Case 1: Config file does not exist
	_, err = Load()
	if err == nil {
		t.Fatal("expected error when config file does not exist, got nil")
	}

	// Case 2: Config file exists but is empty/invalid
	buildDir := filepath.Join(tempHome, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("failed to create .build dir: %v", err)
	}

	configFile := filepath.Join(buildDir, "config.yml")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error when config file is empty, got nil")
	}

	// Case 3: Config file is valid
	validConfig := []byte("agent_adapter: opencode:anthropic/claude-3.5-sonnet")
	if err := os.WriteFile(configFile, validConfig, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading valid config: %v", err)
	}

	if cfg.CLIName != "opencode" || cfg.Provider != "anthropic" || cfg.Model != "claude-3.5-sonnet" {
		t.Errorf("unexpected loaded config: %+v", cfg)
	}
}
