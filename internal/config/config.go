package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the properties parsed from the configuration file.
type Config struct {
	AgentAdapter string // e.g., "opencode:anthropic/claude-3.5-sonnet"
	CLIName      string // e.g., "opencode"
	Provider     string // e.g., "anthropic"
	Model        string // e.g., "claude-3.5-sonnet"
}

// ConfigSource defines the interface for loading raw configuration bytes.
// This abstract source allows the Tester to easily mock file system operations.
type ConfigSource interface {
	Load() ([]byte, error)
}

// FileConfigSource loads configuration data from a specific file path.
type FileConfigSource struct {
	Path string
}

// Load reads the contents of the file at Path.
func (f *FileConfigSource) Load() ([]byte, error) {
	return os.ReadFile(f.Path)
}

// GetConfigPath returns the default absolute path to the user's config file (~/.build/config.yml).
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".build", "config.yml"), nil
}

// Parse parses the config data from an io.Reader.
func Parse(r io.Reader) (*Config, error) {
	scanner := bufio.NewScanner(r)
	var agentAdapter string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Ignore empty lines and comment lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Split by first colon
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "agent_adapter" {
			val := strings.TrimSpace(parts[1])
			agentAdapter = stripCommentsAndQuotes(val)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration: %w", err)
	}

	if agentAdapter == "" {
		return nil, errors.New("missing or empty 'agent_adapter' field in configuration")
	}

	return ParseAdapter(agentAdapter)
}

// ParseBytes parses configuration from a slice of bytes.
func ParseBytes(data []byte) (*Config, error) {
	return Parse(strings.NewReader(string(data)))
}

// ParseAdapter splits a raw agent adapter string into CLI Name and Adapter Name components.
func ParseAdapter(adapter string) (*Config, error) {
	// Expected format: agent:<adapter-name> (e.g., "agent:test_opencode")
	parts := strings.SplitN(adapter, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid agent_adapter format %q: expected format 'agent:adapter_name'", adapter)
	}

	cliName := strings.TrimSpace(parts[0])
	if cliName != "agent" {
		return nil, fmt.Errorf("unsupported CLI %q: only 'agent' is supported", cliName)
	}

	remaining := strings.TrimSpace(parts[1])
	if remaining == "" {
		return nil, fmt.Errorf("invalid agent_adapter format %q: adapter name cannot be empty", adapter)
	}

	return &Config{
		AgentAdapter: adapter,
		CLIName:      "agent",
		Provider:     remaining,
		Model:        "",
	}, nil
}

// stripCommentsAndQuotes cleans quotes and trailing inline comments from the configuration value.
func stripCommentsAndQuotes(val string) string {
	// Strip double quotes if matched
	if strings.HasPrefix(val, "\"") {
		idx := strings.Index(val[1:], "\"")
		if idx != -1 {
			return val[1 : idx+1]
		}
	}
	// Strip single quotes if matched
	if strings.HasPrefix(val, "'") {
		idx := strings.Index(val[1:], "'")
		if idx != -1 {
			return val[1 : idx+1]
		}
	}

	// Strip trailing comment if exists
	if idx := strings.Index(val, "#"); idx != -1 {
		val = val[:idx]
	}

	return strings.TrimSpace(val)
}

// LoadFromSource retrieves configuration using the provided ConfigSource and parses it.
func LoadFromSource(source ConfigSource) (*Config, error) {
	data, err := source.Load()
	if err != nil {
		return nil, err
	}
	return ParseBytes(data)
}

// Load reads and parses the configuration from the default path (~/.build/config.yml).
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromSource(&FileConfigSource{Path: path})
}
