package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
