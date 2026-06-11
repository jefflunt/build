package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jefflunt/build/internal/db"
)


func TestWalkBreakdownDir(t *testing.T) {
	// Setup mock directory
	tempDir, err := os.MkdirTemp("", "breakdown-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create root README.md
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Root Title\nRoot Description"), 0644)

	// Create child directory
	childDir := filepath.Join(tempDir, "child1")
	os.Mkdir(childDir, 0755)
	os.WriteFile(filepath.Join(childDir, "README.md"), []byte("# Child Title\nChild Description"), 0644)
	os.WriteFile(filepath.Join(childDir, "task1.md"), []byte("# Task1 Title\nTask1 Description"), 0644)

	// Run walkBreakdownDir
	node, err := walkBreakdownDir(tempDir, "")
	if err != nil {
		t.Fatal(err)
	}

	if node.Title != "Root Title" {
		t.Errorf("expected Root Title, got %s", node.Title)
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}

	child := node.Children[0]
	if child.Title != "Child Title" {
		t.Errorf("expected Child Title, got %s", child.Title)
	}
	if len(child.Children) != 1 {
		t.Fatalf("expected 1 task in child, got %d", len(child.Children))
	}
	if child.Children[0].Title != "Task1 Title" {
		t.Errorf("expected Task1 Title, got %s", child.Children[0].Title)
	}
}

func TestRunCLI_RM_InvalidSyntax(t *testing.T) {
	if os.Getenv("BE_CRASHER_RM_INVALID") == "1" {
		runCLI([]string{"build", "rm"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_InvalidSyntax")
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_INVALID=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit status 1 for invalid rm syntax, got 0")
	}
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "Usage: build rm <id:<task-id>|status:<status>>") {
		t.Errorf("expected usage message, got: %q", out)
	}
}

func TestRunCLI_RM_NoMatches(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize an empty database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_VALID") == "1" {
		runCLI([]string{"build", "rm", "status:nonexistent"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_NoMatches")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_VALID=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "No tasks found matching the criteria.") {
		t.Errorf("expected 'No tasks found' message, got: %q", out)
	}
}

func TestRunCLI_RM_Success(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-success-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	// Insert test data
	_, err = database.Exec(`
		INSERT INTO tasks (id, status) VALUES ('task1', 'todo');
		INSERT INTO comments (task_id, content) VALUES ('task1', 'some comment');
		INSERT INTO audit_logs (task_id, action) VALUES ('task1', 'some action');
	`)
	if err != nil {
		database.Close()
		t.Fatalf("Failed to insert mock data: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_SUCCESS") == "1" {
		runCLI([]string{"build", "rm", "id:task1"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_Success")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_SUCCESS=1")
	cmd.Stdin = strings.NewReader("y\n") // Simulate confirming deletion

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Successfully deleted 1 tasks.") {
		t.Errorf("expected success message, got: %q", out)
	}

	// Verify tasks are deleted from tables
	dbVerify, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB for verification: %v", err)
	}
	defer dbVerify.Close()

	var count int
	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected task1 to be deleted from tasks table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected comments to be deleted from comments table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Expected audit logs to be deleted from audit_logs table")
	}
}

func TestRunCLI_RM_Abort(t *testing.T) {
	// Setup a temporary directory to run the test in so we have a clean/isolated environment
	tempDir, err := os.MkdirTemp("", "main-rm-abort-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create .build directory and initialize database
	buildDir := filepath.Join(tempDir, ".build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(buildDir, "build.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	// Insert test data
	_, err = database.Exec(`
		INSERT INTO tasks (id, status) VALUES ('task1', 'todo');
		INSERT INTO comments (task_id, content) VALUES ('task1', 'some comment');
		INSERT INTO audit_logs (task_id, action) VALUES ('task1', 'some action');
	`)
	if err != nil {
		database.Close()
		t.Fatalf("Failed to insert mock data: %v", err)
	}
	database.Close()

	if os.Getenv("BE_CRASHER_RM_ABORT") == "1" {
		runCLI([]string{"build", "rm", "id:task1"})
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCLI_RM_Abort")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "BE_CRASHER_RM_ABORT=1")
	cmd.Stdin = strings.NewReader("n\n") // Simulate aborting deletion

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Deletion aborted.") {
		t.Errorf("expected abort message, got: %q", out)
	}

	// Verify tasks are NOT deleted from tables
	dbVerify, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB for verification: %v", err)
	}
	defer dbVerify.Close()

	var count int
	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected task1 NOT to be deleted from tasks table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected comments NOT to be deleted from comments table")
	}

	_ = dbVerify.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Expected audit logs NOT to be deleted from audit_logs table")
	}
}

