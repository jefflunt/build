package timecmd

import (
	"database/sql"
	"testing"

	_ "github.com/glebarez/go-sqlite"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero", 0, "0h00m00s"},
		{"seconds only", 45, "0h00m45s"},
		{"minutes and seconds", 65, "0h01m05s"},
		{"exact hour", 3600, "1h00m00s"},
		{"hour min sec", 3665, "1h01m05s"},
		{"multiple hours", 7325, "2h02m05s"},
		{"large hours", 360000, "100h00m00s"},
		{"negative", -10, "0h00m00s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("FormatDuration(%d) = %s; expected %s", tt.seconds, result, tt.expected)
			}
		})
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	schema := `
	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		parent_id TEXT,
		title TEXT,
		status TEXT
	);
	CREATE TABLE audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		duration_seconds INTEGER
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	insertTasks := `
	INSERT INTO tasks (id, parent_id, title, status) VALUES 
	('root', NULL, 'Root Task', 'done'),
	('child1', 'root', 'Child 1', 'done'),
	('child2', 'root', 'Child 2', 'todo'),
	('grandchild1', 'child2', 'Grandchild 1', 'done'),
	('child3', 'root', 'Child 3', 'done');
	`
	_, err = db.Exec(insertTasks)
	if err != nil {
		t.Fatalf("Failed to insert tasks: %v", err)
	}

	insertLogs := `
	INSERT INTO audit_logs (task_id, duration_seconds) VALUES
	('root', 5),
	('root', 5),
	('child1', 20),
	('child2', 30),
	('grandchild1', 40),
	('child3', 25),
	('child3', 25);
	`
	_, err = db.Exec(insertLogs)
	if err != nil {
		t.Fatalf("Failed to insert audit logs: %v", err)
	}

	return db
}

func TestBuildTimeTree(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	roots, err := BuildTimeTree(db)
	if err != nil {
		t.Fatalf("BuildTimeTree returned error: %v", err)
	}

	if len(roots) != 2 {
		t.Fatalf("Expected 2 roots (root, grandchild1), got %d", len(roots))
	}

	if roots[0].ID != "grandchild1" || roots[1].ID != "root" {
		t.Errorf("Unexpected roots or sort order. 0: %s, 1: %s", roots[0].ID, roots[1].ID)
	}

	if roots[0].DirectTime != 40 || roots[0].TotalTime != 40 {
		t.Errorf("Grandchild1 times incorrect. Direct: %d, Total: %d", roots[0].DirectTime, roots[0].TotalTime)
	}

	rootNode := roots[1]
	if rootNode.DirectTime != 10 {
		t.Errorf("Root direct time incorrect. Expected 10, got %d", rootNode.DirectTime)
	}
	if rootNode.TotalTime != 80 {
		t.Errorf("Root total time incorrect. Expected 80, got %d", rootNode.TotalTime)
	}

	if len(rootNode.Children) != 2 {
		t.Fatalf("Expected root to have 2 children, got %d", len(rootNode.Children))
	}

	if rootNode.Children[0].ID != "child1" || rootNode.Children[1].ID != "child3" {
		t.Errorf("Unexpected children or sort order")
	}

	if rootNode.Children[1].DirectTime != 50 || rootNode.Children[1].TotalTime != 50 {
		t.Errorf("Child3 times incorrect. Direct: %d, Total: %d", rootNode.Children[1].DirectTime, rootNode.Children[1].TotalTime)
	}
}