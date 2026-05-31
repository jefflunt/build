package rmcmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"strings"
)

// DB defines the database operations required for rmcmd.
// This interface separates I/O from the business logic of recursively resolving IDs.
type DB interface {
	TaskExists(id string) (bool, error)
	GetTasksByStatus(status string) ([]string, error)
	GetChildrenIDs(parentIDs []string) ([]string, error)
	DeleteTasks(ids []string) error
}

// SQLDB implements the DB interface using a real sql.DB connection.
type SQLDB struct {
	db *sql.DB
}

// NewSQLDB creates a new SQLDB instance.
func NewSQLDB(db *sql.DB) *SQLDB {
	return &SQLDB{db: db}
}

// TaskExists checks if the task exists in the tasks table.
func (s *SQLDB) TaskExists(id string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetTasksByStatus returns all task IDs that match the given status.
func (s *SQLDB) GetTasksByStatus(status string) ([]string, error) {
	rows, err := s.db.Query("SELECT id FROM tasks WHERE status = ?", status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// GetChildrenIDs returns all direct child task IDs for the given parent IDs.
func (s *SQLDB) GetChildrenIDs(parentIDs []string) ([]string, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(parentIDs))
	args := make([]interface{}, len(parentIDs))
	for i, id := range parentIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id FROM tasks WHERE parent_id IN (%s)", strings.Join(placeholders, ","))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// DeleteTasks safely purges records from the audit_logs, comments, and tasks tables.
func (s *SQLDB) DeleteTasks(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	inClause := strings.Join(placeholders, ",")

	// Delete from audit_logs
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM audit_logs WHERE task_id IN (%s)", inClause), args...)
	if err != nil {
		return fmt.Errorf("failed to delete from audit_logs: %w", err)
	}

	// Delete from comments
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM comments WHERE task_id IN (%s)", inClause), args...)
	if err != nil {
		return fmt.Errorf("failed to delete from comments: %w", err)
	}

	// Delete from tasks
	_, err = tx.Exec(fmt.Sprintf("DELETE FROM tasks WHERE id IN (%s)", inClause), args...)
	if err != nil {
		return fmt.Errorf("failed to delete from tasks: %w", err)
	}

	return tx.Commit()
}

// ResolveDescendantsOfID recursively collects the target task ID and all its descendants.
func ResolveDescendantsOfID(db DB, targetID string) ([]string, error) {
	exists, err := db.TaskExists(targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to check task existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("task ID %s not found", targetID)
	}

	return resolve(db, []string{targetID})
}

// ResolveDescendantsOfStatus recursively collects all tasks matching the status and all their descendants.
func ResolveDescendantsOfStatus(db DB, status string) ([]string, error) {
	startingIDs, err := db.GetTasksByStatus(status)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by status: %w", err)
	}
	if len(startingIDs) == 0 {
		return nil, nil
	}

	return resolve(db, startingIDs)
}

func resolve(db DB, startingIDs []string) ([]string, error) {
	visited := make(map[string]bool)
	var allIDs []string

	toProcess := append([]string{}, startingIDs...)
	for _, id := range startingIDs {
		if !visited[id] {
			visited[id] = true
			allIDs = append(allIDs, id)
		}
	}

	for len(toProcess) > 0 {
		children, err := db.GetChildrenIDs(toProcess)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve child tasks: %w", err)
		}

		var nextToProcess []string
		for _, child := range children {
			if !visited[child] {
				visited[child] = true
				allIDs = append(allIDs, child)
				nextToProcess = append(nextToProcess, child)
			}
		}
		toProcess = nextToProcess
	}

	return allIDs, nil
}

// ResolveTasksToDelete parses the target (e.g. "id:G1" or "status:done") and returns the complete set of unique task IDs to be deleted.
func ResolveTasksToDelete(db DB, target string) ([]string, error) {
	if strings.HasPrefix(target, "id:") {
		id := strings.TrimPrefix(target, "id:")
		if id == "" {
			return nil, fmt.Errorf("task ID cannot be empty")
		}
		return ResolveDescendantsOfID(db, id)
	} else if strings.HasPrefix(target, "status:") {
		status := strings.TrimPrefix(target, "status:")
		if status == "" {
			return nil, fmt.Errorf("status cannot be empty")
		}
		return ResolveDescendantsOfStatus(db, status)
	}
	return nil, fmt.Errorf("invalid target format, must be 'id:<task-id>' or 'status:<status>'")
}

// PromptConfirmation asks the user for confirmation via the provided io.Reader and io.Writer.
func PromptConfirmation(r io.Reader, w io.Writer, taskCount int) (bool, error) {
	fmt.Fprintf(w, "This will delete %d tasks (including all descendants, comments, and audit logs).\n", taskCount)
	fmt.Fprintf(w, "Are you sure you want to proceed? [y/N]: ")

	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if strings.ToLower(text) == "y" {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// ExecuteRM handles the complete execution of the rm command: resolving, prompting, and deleting.
func ExecuteRM(db DB, r io.Reader, w io.Writer, target string) error {
	ids, err := ResolveTasksToDelete(db, target)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		fmt.Fprintln(w, "No tasks found matching the criteria.")
		return nil
	}

	confirmed, err := PromptConfirmation(r, w, len(ids))
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if !confirmed {
		fmt.Fprintln(w, "Deletion aborted.")
		return nil
	}

	if err := db.DeleteTasks(ids); err != nil {
		return fmt.Errorf("failed to delete tasks: %w", err)
	}

	fmt.Fprintf(w, "Successfully deleted %d tasks.\n", len(ids))
	return nil
}
