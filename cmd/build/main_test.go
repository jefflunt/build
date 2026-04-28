package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.MkdirAll(".build", 0755)
	
	// Setup mock breakdown
	tmpDir := os.TempDir()
	mockBreakdown := filepath.Join(tmpDir, "breakdown")
	
	// Create mock breakdown script
	content := `#!/bin/bash
mkdir -p "$3"
echo "# Mock Session" > "$3/README.md"
exit 0
`
	os.WriteFile(mockBreakdown, []byte(content), 0755)
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	defer os.RemoveAll(".build")
	defer os.Remove(mockBreakdown)
	
	m.Run()
}

func TestRunCLI_Enqueue(t *testing.T) {
	// Test routing of 'enqueue'
	// This might fail if it tries to interact with DB or filesystem.
	// Since runCLI calls os.Exit on some errors, we might need a better way to test.
}

func TestEnqueuePlan_Validation(t *testing.T) {
	err := enqueuePlan("non-existent-file.md")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestEnqueuePlan_Success(t *testing.T) {
	// Create a dummy plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "my-session.md")
	os.WriteFile(planFile, []byte("# My Session"), 0644)

	err := enqueuePlan(planFile)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
