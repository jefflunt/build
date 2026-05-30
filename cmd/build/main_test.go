package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLLMConfig(t *testing.T) {
	// Setup: unset env vars
	os.Unsetenv("BUILD_LLM_PROVIDER")
	os.Unsetenv("BUILD_LLM_MODEL")

	var buf bytes.Buffer
	code := validateLLMConfig(&buf)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}

	if buf.Len() == 0 {
		t.Error("expected output, got none")
	}

	// Setup: set env vars
	os.Setenv("BUILD_LLM_PROVIDER", "google")
	os.Setenv("BUILD_LLM_MODEL", "gemini-pro")

	buf.Reset()
	code = validateLLMConfig(&buf)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if buf.Len() != 0 {
		t.Errorf("expected no output, got: %s", buf.String())
	}
}

func TestGetValidModels(t *testing.T) {
	// Create a temporary directory for the mock command
	tempDir, err := os.MkdirTemp("", "mock-opencode")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock opencode script
	mockOpencodePath := filepath.Join(tempDir, "opencode")

	// Create a script that prints mock models and exits 0
	scriptContent := `#!/usr/bin/env bash
echo "gemini-1.5-pro"
echo "gemini-1.5-flash"
`
	if err := os.WriteFile(mockOpencodePath, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	// Manipulate PATH to use our mock
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)

	// Run the function
	models := getValidModels()

	// Verify the results
	expected := []string{"gemini-1.5-pro", "gemini-1.5-flash"}
	if len(models) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, models)
	}
	for i := range expected {
		if models[i] != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], models[i])
		}
	}
}

func TestGetValidModelsFallback(t *testing.T) {
	// Ensure opencode is not in PATH or fails
	// We can set PATH to an empty string to ensure opencode is not found
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", "")

	// Run the function
	models := getValidModels()

	// Verify the fallback results
	expected := []string{"gemini-3.1-flash-lite-preview", "gemini-3.1-pro", "gpt-4o"}
	if len(models) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, models)
	}
	for i := range expected {
		if models[i] != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], models[i])
		}
	}
}
