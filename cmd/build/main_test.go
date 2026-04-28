package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"github.com/jefflunt/build/internal/db"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.MkdirAll(".build", 0755)
	database, _ := db.InitDB(".build/build.db")
	database.Close()
	
	// Setup mock breakdown
	tmpDir := os.TempDir()
	mockBreakdown := filepath.Join(tmpDir, "breakdown")
	
	// Create mock breakdown script
	content := `#!/bin/bash
echo "$1 $2 $3" > .breakdown_args
mkdir -p "$3"
echo "# Mock Session" > "$3/README.md"
echo "# Task 1" > "$3/task1.md"
exit 0
`
	os.WriteFile(mockBreakdown, []byte(content), 0755)
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	code := m.Run()

	os.RemoveAll(".build")
	os.Remove(mockBreakdown)
	os.Exit(code)
}

func TestRunCLI_Enqueue(t *testing.T) {
	// Create a dummy plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "my-session.md")
	os.WriteFile(planFile, []byte("# My Session"), 0644)

	cmd := exec.Command("go", "run", ".", "enqueue", planFile)
	if err := cmd.Run(); err != nil {
		t.Errorf("expected no error from CLI, got %v", err)
	}
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

	// Clean previous args
	os.Remove(".breakdown_args")

	err := enqueuePlan(planFile)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify arguments passed to mock breakdown
	args, err := os.ReadFile(".breakdown_args")
	if err != nil {
		t.Fatalf("failed to read .breakdown_args: %v", err)
	}
	
	sessionName := "my-session"
	expectedDir := filepath.Join("/tmp", "build", "breakdowns", sessionName)
	expectedArgs := "-v " + planFile + " " + expectedDir
	if strings.TrimSpace(string(args)) != expectedArgs {
		t.Errorf("expected args %q, got %q", expectedArgs, string(args))
	}

	// Verify database was populated
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		t.Errorf("failed to init DB: %v", err)
	}
	defer database.Close()
	
	var title string
	err = database.QueryRow("SELECT title FROM tasks WHERE title LIKE 'Task 1'").Scan(&title)
	if err != nil {
		t.Errorf("expected task 'Task 1' in DB, got error: %v", err)
	}
}
