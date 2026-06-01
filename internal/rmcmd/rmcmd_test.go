package rmcmd

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	schema := `
	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		parent_id TEXT,
		status TEXT DEFAULT 'todo'
	);
	CREATE TABLE audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT
	);
	CREATE TABLE comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestSQLDB_TaskExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, status) VALUES ('task1', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	exists, err := s.TaskExists("task1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !exists {
		t.Error("Expected task1 to exist")
	}

	exists, err = s.TaskExists("nonexistent")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exists {
		t.Error("Expected nonexistent to not exist")
	}
}

func TestSQLDB_GetTasksByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, status) VALUES 
		('t1', 'todo'),
		('t2', 'done'),
		('t3', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	ids, err := s.GetTasksByStatus("todo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "t1" || ids[1] != "t3" {
		t.Errorf("Unexpected tasks: %v", ids)
	}

	ids, err = s.GetTasksByStatus("in_progress")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("Expected 0 tasks, got: %v", ids)
	}
}

func TestSQLDB_GetChildrenIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, parent_id) VALUES 
		('parent1', NULL),
		('child1', 'parent1'),
		('child2', 'parent1'),
		('parent2', NULL),
		('child3', 'parent2')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	// Test with multiple parents
	children, err := s.GetChildrenIDs([]string{"parent1", "parent2"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := map[string]bool{"child1": true, "child2": true, "child3": true}
	if len(children) != 3 {
		t.Errorf("Expected 3 children, got %v", children)
	}
	for _, c := range children {
		if !expected[c] {
			t.Errorf("Unexpected child: %s", c)
		}
	}

	// Test with empty parents list
	children, err = s.GetChildrenIDs([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("Expected 0 children, got %v", children)
	}
}

func TestSQLDB_DeleteTasks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO tasks (id) VALUES ('t1'), ('t2');
		INSERT INTO comments (task_id) VALUES ('t1'), ('t2'), ('t1');
		INSERT INTO audit_logs (task_id) VALUES ('t1'), ('t2');
	`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	err = s.DeleteTasks([]string{"t1"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify t1 is deleted, but t2 is not
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 't1'").Scan(&count)
	if count != 0 {
		t.Error("t1 task was not deleted")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 't2'").Scan(&count)
	if count != 1 {
		t.Error("t2 task was deleted")
	}

	_ = db.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 't1'").Scan(&count)
	if count != 0 {
		t.Error("t1 comments were not deleted")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = 't2'").Scan(&count)
	if count != 1 {
		t.Error("t2 comment was deleted")
	}

	_ = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 't1'").Scan(&count)
	if count != 0 {
		t.Error("t1 audit logs were not deleted")
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE task_id = 't2'").Scan(&count)
	if count != 1 {
		t.Error("t2 audit log was deleted")
	}

	// Delete empty slice should do nothing
	err = s.DeleteTasks([]string{})
	if err != nil {
		t.Errorf("Unexpected error on empty slice: %v", err)
	}
}

func TestResolveDescendantsOfID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, parent_id, status) VALUES 
		('root', NULL, 'todo'),
		('child1', 'root', 'todo'),
		('child2', 'root', 'done'),
		('grandchild1', 'child1', 'todo'),
		('other', NULL, 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	// Resolve from 'root' - should return root, child1, child2, grandchild1
	ids, err := ResolveDescendantsOfID(s, "root")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := map[string]bool{"root": true, "child1": true, "child2": true, "grandchild1": true}
	if len(ids) != 4 {
		t.Errorf("Expected 4 IDs, got %v", ids)
	}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("Unexpected resolved ID: %s", id)
		}
	}

	// Resolve from non-existent ID
	_, err = ResolveDescendantsOfID(s, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent target ID")
	} else if !strings.Contains(err.Error(), "task ID nonexistent not found") {
		t.Errorf("Expected 'not found' error message, got: %v", err)
	}
}

func TestResolveDescendantsOfStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, parent_id, status) VALUES 
		('root1', NULL, 'todo'),
		('child1', 'root1', 'done'), 
		('root2', NULL, 'done'),
		('child2', 'root2', 'todo'), 
		('grandchild2', 'child2', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	// Resolve 'todo' status:
	// Start with 'root1' (todo) and 'child2' (todo) and 'grandchild2' (todo)
	// 'root1' -> resolves 'child1' (descendant)
	// 'child2' -> resolves 'grandchild2'
	// 'grandchild2' -> no children
	// Unique expected set: 'root1', 'child1', 'child2', 'grandchild2'
	ids, err := ResolveDescendantsOfStatus(s, "todo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expected := map[string]bool{"root1": true, "child1": true, "child2": true, "grandchild2": true}
	if len(ids) != 4 {
		t.Errorf("Expected 4 IDs, got %v", ids)
	}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("Unexpected resolved ID: %s", id)
		}
	}

	// Resolve non-existent status
	ids, err = ResolveDescendantsOfStatus(s, "nonexistent")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("Expected empty result, got %v", ids)
	}
}

func TestResolveTasksToDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, parent_id, status) VALUES 
		('root', NULL, 'todo'),
		('child', 'root', 'done')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	// Parse 'id:root'
	ids, err := ResolveTasksToDelete(s, "id:root")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %v", ids)
	}

	// Parse 'status:todo'
	ids, err = ResolveTasksToDelete(s, "status:todo")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %v", ids)
	}

	// Error cases
	if _, err := ResolveTasksToDelete(s, "id:"); err == nil {
		t.Error("Expected error on empty ID")
	}
	if _, err := ResolveTasksToDelete(s, "status:"); err == nil {
		t.Error("Expected error on empty status")
	}
	if _, err := ResolveTasksToDelete(s, "invalid:format"); err == nil {
		t.Error("Expected error on invalid target format")
	}
}

func TestPromptConfirmation(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"y", true},
		{"n\n", false},
		{"yes\n", false},
		{"\n", false},
		{"anything\n", false},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			var buf bytes.Buffer
			in := strings.NewReader(tt.input)
			res, err := PromptConfirmation(in, &buf, 5)
			if err != nil {
				t.Fatalf("PromptConfirmation error: %v", err)
			}
			if res != tt.expected {
				t.Errorf("PromptConfirmation(%q) = %v; expected %v", tt.input, res, tt.expected)
			}
			if !strings.Contains(buf.String(), "This will delete 5 tasks") {
				t.Errorf("Expected prompt message to contain task count, got: %s", buf.String())
			}
		})
	}
}

type MockDB struct {
	TaskExistsFunc       func(id string) (bool, error)
	GetTasksByStatusFunc func(status string) ([]string, error)
	GetChildrenIDsFunc   func(parentIDs []string) ([]string, error)
	DeleteTasksFunc      func(ids []string) error
}

func (m *MockDB) TaskExists(id string) (bool, error) {
	if m.TaskExistsFunc != nil {
		return m.TaskExistsFunc(id)
	}
	return false, nil
}

func (m *MockDB) GetTasksByStatus(status string) ([]string, error) {
	if m.GetTasksByStatusFunc != nil {
		return m.GetTasksByStatusFunc(status)
	}
	return nil, nil
}

func (m *MockDB) GetChildrenIDs(parentIDs []string) ([]string, error) {
	if m.GetChildrenIDsFunc != nil {
		return m.GetChildrenIDsFunc(parentIDs)
	}
	return nil, nil
}

func (m *MockDB) DeleteTasks(ids []string) error {
	if m.DeleteTasksFunc != nil {
		return m.DeleteTasksFunc(ids)
	}
	return nil
}

func TestExecuteRM_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, status) VALUES ('task1', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	var w bytes.Buffer
	r := strings.NewReader("y\n")

	err = ExecuteRM(s, r, &w, "id:task1")
	if err != nil {
		t.Fatalf("ExecuteRM error: %v", err)
	}

	output := w.String()
	if !strings.Contains(output, "Successfully deleted 1 tasks.") {
		t.Errorf("Unexpected output: %q", output)
	}

	// Verify task deleted
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("Task was not deleted")
	}
}

func TestExecuteRM_Abort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO tasks (id, status) VALUES ('task1', 'todo')`)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	s := NewSQLDB(db)

	var w bytes.Buffer
	r := strings.NewReader("n\n")

	err = ExecuteRM(s, r, &w, "id:task1")
	if err != nil {
		t.Fatalf("ExecuteRM error: %v", err)
	}

	output := w.String()
	if !strings.Contains(output, "Deletion aborted.") {
		t.Errorf("Unexpected output: %q", output)
	}

	// Verify task not deleted
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 1 {
		t.Error("Task was deleted when it shouldn't have been")
	}
}

func TestExecuteRM_NoMatches(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewSQLDB(db)

	var w bytes.Buffer
	r := strings.NewReader("y\n")

	err := ExecuteRM(s, r, &w, "status:done")
	if err != nil {
		t.Fatalf("ExecuteRM error: %v", err)
	}

	output := w.String()
	if !strings.Contains(output, "No tasks found matching the criteria.") {
		t.Errorf("Unexpected output: %q", output)
	}
}

func TestExecuteRM_ResolveError(t *testing.T) {
	mock := &MockDB{
		TaskExistsFunc: func(id string) (bool, error) {
			return false, errors.New("db error")
		},
	}

	var w bytes.Buffer
	r := strings.NewReader("y\n")

	err := ExecuteRM(mock, r, &w, "id:task1")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "db error") {
		t.Errorf("Expected 'db error', got: %v", err)
	}
}

func TestExecuteRM_DeleteError(t *testing.T) {
	mock := &MockDB{
		TaskExistsFunc: func(id string) (bool, error) {
			return true, nil
		},
		DeleteTasksFunc: func(ids []string) error {
			return errors.New("delete error")
		},
	}

	var w bytes.Buffer
	r := strings.NewReader("y\n")

	err := ExecuteRM(mock, r, &w, "id:task1")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete error") {
		t.Errorf("Expected 'delete error', got: %v", err)
	}
}
