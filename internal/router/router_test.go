package router

import (
	"context"
	"embed"
	"io"
	"os"
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
