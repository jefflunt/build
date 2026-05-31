package router

import (
	"context"
	"database/sql"
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jefflunt/build/internal/db"
)

//go:embed templates/*.md
var testTemplates embed.FS

// mockClient implements cli.Client interface for testing
type mockClient struct {
	runFunc    func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error
	modelsFunc func(ctx context.Context) ([]string, error)
}

func (m *mockClient) Run(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, model, agent, prompt, stdout, stderr)
	}
	return nil
}

func (m *mockClient) Models(ctx context.Context) ([]string, error) {
	if m.modelsFunc != nil {
		return m.modelsFunc(ctx)
	}
	return nil, nil
}

func TestRouter_Reconcile(t *testing.T) {
	// Keep track of original directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create safe isolated temp directory
	tempDir, err := os.MkdirTemp("", "router-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create .build folder (needed by agent file writes)
	err = os.MkdirAll(".build", 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Configure mock git env vars so git commit doesn't fail
	os.Setenv("GIT_AUTHOR_NAME", "Tester")
	os.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "Tester")
	os.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	defer func() {
		os.Unsetenv("GIT_AUTHOR_NAME")
		os.Unsetenv("GIT_AUTHOR_EMAIL")
		os.Unsetenv("GIT_COMMITTER_NAME")
		os.Unsetenv("GIT_COMMITTER_EMAIL")
	}()

	// Initialize in-memory SQLite database
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}
	defer database.Close()

	// Seed agents
	_, err = database.Exec(`
		INSERT INTO agents (id, role, name) VALUES 
		(1, 'owner', 'Owner'),
		(2, 'dev', 'Developer'),
		(3, 'tester', 'Tester'),
		(4, 'boss', 'Boss'),
		(5, 'lead', 'Lead Engineer'),
		(6, 'sweep', 'Git Cleanup Artist')
	`)
	if err != nil {
		t.Fatalf("failed to seed agents: %v", err)
	}

	// Create a task
	_, err = database.Exec(`
		INSERT INTO tasks (id, parent_id, type, title, description, status, agent_id) VALUES 
		('T1', NULL, 'task', 'Implement feature X', 'Description of feature X', 'todo', 2)
	`)
	if err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	// Define mock CLI client
	type call struct {
		model, agent, prompt string
	}
	var calls []call
	mockCli := &mockClient{
		runFunc: func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
			calls = append(calls, call{model: model, agent: agent, prompt: prompt})
			return nil
		},
	}

	// Create Router using the CLI client interface!
	r := NewRouter(database, mockCli, "google", "gemini-pro", testTemplates)

	// Run reconcile
	err = r.reconcile()
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// Verify CLI run was called with correct parameters!
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 mock CLI Run calls, got %d", len(calls))
	}

	// First call should be processing the dev task
	devCall := calls[0]
	if devCall.model != "google/gemini-pro" {
		t.Errorf("expected first call model 'google/gemini-pro', got %q", devCall.model)
	}
	if devCall.agent != "build" {
		t.Errorf("expected first call agent 'build', got %q", devCall.agent)
	}
	if !strings.Contains(devCall.prompt, "Implement feature X") {
		t.Errorf("expected first call prompt to contain task title, got %q", devCall.prompt)
	}

	// Second call should be for sweep agent
	sweepCall := calls[1]
	if sweepCall.model != "google/gemini-pro" {
		t.Errorf("expected second call model 'google/gemini-pro', got %q", sweepCall.model)
	}
	if sweepCall.agent != "build" {
		t.Errorf("expected second call agent 'build', got %q", sweepCall.agent)
	}
	if sweepCall.prompt != "dummy sweep template" {
		t.Errorf("expected second call prompt to be 'dummy sweep template', got %q", sweepCall.prompt)
	}

	// Verify post-session logic transitioned assignee to Tester (3)
	var agentID int
	err = database.QueryRow("SELECT agent_id FROM tasks WHERE id = 'T1'").Scan(&agentID)
	if err != nil {
		t.Fatalf("failed to query task assignee: %v", err)
	}
	if agentID != 3 {
		t.Errorf("expected assignee to transition to 3 (Tester), got %d", agentID)
	}
}

func setupRouterTest(t *testing.T) (origDir, tempDir string, database *sql.DB, mockCli *mockClient) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err = os.MkdirTemp("", "router-test")
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}

	err = os.MkdirAll(".build", 0755)
	if err != nil {
		os.Chdir(origDir)
		os.RemoveAll(tempDir)
		t.Fatal(err)
	}

	os.Setenv("GIT_AUTHOR_NAME", "Tester")
	os.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "Tester")
	os.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")

	database, err = db.InitDB(":memory:")
	if err != nil {
		os.Chdir(origDir)
		os.RemoveAll(tempDir)
		t.Fatalf("failed to init database: %v", err)
	}

	_, err = database.Exec(`
		INSERT INTO agents (id, role, name) VALUES 
		(1, 'owner', 'Owner'),
		(2, 'dev', 'Developer'),
		(3, 'tester', 'Tester'),
		(4, 'boss', 'Boss'),
		(5, 'lead', 'Lead Engineer'),
		(6, 'sweep', 'Git Cleanup Artist')
	`)
	if err != nil {
		database.Close()
		os.Chdir(origDir)
		os.RemoveAll(tempDir)
		t.Fatalf("failed to seed agents: %v", err)
	}

	mockCli = &mockClient{}

	return origDir, tempDir, database, mockCli
}

func teardownRouterTest(origDir, tempDir string, database *sql.DB) {
	if database != nil {
		database.Close()
	}
	os.Chdir(origDir)
	os.RemoveAll(tempDir)
	os.Unsetenv("GIT_AUTHOR_NAME")
	os.Unsetenv("GIT_AUTHOR_EMAIL")
	os.Unsetenv("GIT_COMMITTER_NAME")
	os.Unsetenv("GIT_COMMITTER_EMAIL")
}

func TestRouter_Reconcile_BlockedByFailedTask(t *testing.T) {
	origDir, tempDir, database, mockCli := setupRouterTest(t)
	defer teardownRouterTest(origDir, tempDir, database)

	// Seed a failed task
	_, err := database.Exec(`
		INSERT INTO tasks (id, parent_id, type, title, description, status, agent_id) VALUES 
		('T1', NULL, 'task', 'Implement feature X', 'Description of feature X', 'failed', 2)
	`)
	if err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	var runCalled bool
	mockCli.runFunc = func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
		runCalled = true
		return nil
	}

	r := NewRouter(database, mockCli, "google", "gemini-pro", testTemplates)
	err = r.reconcile()
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	if runCalled {
		t.Error("expected mock CLI Run NOT to be called when blocked by failed task")
	}

	// Verify status remains failed and assignee is still 2
	var status string
	var agentID int
	err = database.QueryRow("SELECT status, agent_id FROM tasks WHERE id = 'T1'").Scan(&status, &agentID)
	if err != nil {
		t.Fatal(err)
	}
	if status != "failed" || agentID != 2 {
		t.Errorf("expected status 'failed' and assignee 2, got %q and %d", status, agentID)
	}
}

func TestRouter_Reconcile_StuckTaskEscalatedToLead(t *testing.T) {
	origDir, tempDir, database, mockCli := setupRouterTest(t)
	defer teardownRouterTest(origDir, tempDir, database)

	// Seed a stuck task (assignee is Lead 5)
	_, err := database.Exec(`
		INSERT INTO tasks (id, parent_id, type, title, description, status, agent_id) VALUES 
		('T1', NULL, 'task', 'Stuck task', 'Description', 'stuck', 5)
	`)
	if err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	var calls []string
	mockCli.runFunc = func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
		calls = append(calls, agent)
		return nil
	}

	r := NewRouter(database, mockCli, "google", "gemini-pro", testTemplates)
	err = r.reconcile()
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// It should call client.Run for the Lead task, and then for the sweep agent
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 run calls, got %d", len(calls))
	}
	if calls[0] != "build" {
		t.Errorf("expected first call to agent 'build', got %q", calls[0])
	}

	// Under handlePostSession, Lead finished transitions back to Dev (agent_id = 2, status = 'todo')
	var status string
	var agentID int
	err = database.QueryRow("SELECT status, agent_id FROM tasks WHERE id = 'T1'").Scan(&status, &agentID)
	if err != nil {
		t.Fatal(err)
	}
	if status != "todo" || agentID != 2 {
		t.Errorf("expected task to transition back to 'todo' with assignee 2 (dev) after Lead finished, got status %q and assignee %d", status, agentID)
	}
}

func TestRouter_Reconcile_TesterFinished_Success(t *testing.T) {
	origDir, tempDir, database, mockCli := setupRouterTest(t)
	defer teardownRouterTest(origDir, tempDir, database)

	// Seed a task with Tester assignee (3)
	_, err := database.Exec(`
		INSERT INTO tasks (id, parent_id, type, title, description, status, agent_id) VALUES 
		('T1', NULL, 'task', 'Test Task', 'Description', 'todo', 3)
	`)
	if err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	// Create a successful test script adapter
	testScript := filepath.Join(".build", "test")
	if err := os.WriteFile(testScript, []byte("#!/usr/bin/env bash\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write mock test script: %v", err)
	}

	mockCli.runFunc = func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
		return nil
	}

	r := NewRouter(database, mockCli, "google", "gemini-pro", testTemplates)
	err = r.reconcile()
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// Verify task transitions to assignee 4 (Boss)
	var agentID int
	err = database.QueryRow("SELECT agent_id FROM tasks WHERE id = 'T1'").Scan(&agentID)
	if err != nil {
		t.Fatal(err)
	}
	if agentID != 4 {
		t.Errorf("expected assignee to transition to 4 (Boss) on successful tests, got %d", agentID)
	}
}

func TestRouter_Reconcile_TesterFinished_Failure(t *testing.T) {
	origDir, tempDir, database, mockCli := setupRouterTest(t)
	defer teardownRouterTest(origDir, tempDir, database)

	// Seed a task with Tester assignee (3)
	_, err := database.Exec(`
		INSERT INTO tasks (id, parent_id, type, title, description, status, agent_id) VALUES 
		('T1', NULL, 'task', 'Test Task', 'Description', 'todo', 3)
	`)
	if err != nil {
		t.Fatalf("failed to seed tasks: %v", err)
	}

	// Create a failing test script adapter
	testScript := filepath.Join(".build", "test")
	if err := os.WriteFile(testScript, []byte("#!/usr/bin/env bash\nexit 1\n"), 0755); err != nil {
		t.Fatalf("failed to write mock test script: %v", err)
	}

	mockCli.runFunc = func(ctx context.Context, model, agent, prompt string, stdout, stderr io.Writer) error {
		return nil
	}

	r := NewRouter(database, mockCli, "google", "gemini-pro", testTemplates)
	err = r.reconcile()
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// Verify task transitions back to assignee 2 (Dev)
	var agentID int
	err = database.QueryRow("SELECT agent_id FROM tasks WHERE id = 'T1'").Scan(&agentID)
	if err != nil {
		t.Fatal(err)
	}
	if agentID != 2 {
		t.Errorf("expected assignee to transition back to 2 (Dev) on failing tests, got %d", agentID)
	}

	// Verify that a comment was added detailing the test failure
	var commentCount int
	err = database.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 'T1'").Scan(&commentCount)
	if err != nil {
		t.Fatal(err)
	}
	if commentCount == 0 {
		t.Error("expected a comment to be created on task T1 detailing the test failure")
	}
}
