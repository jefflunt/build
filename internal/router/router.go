package router

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Router represents the persistent background service.
type Router struct {
	db *sql.DB
}

// NewRouter creates a new router instance.
func NewRouter(db *sql.DB) *Router {
	return &Router{db: db}
}

// Run starts the persistent reconciliation loop.
func (r *Router) Run() error {
	fmt.Println("Router service started...")
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		if err := r.reconcile(); err != nil {
			fmt.Printf("Error reconciling: %v\n", err)
		}
	}
	return nil
}

func (r *Router) reconcile() error {
	// 1. Check if blocked by ANY failed task
	var failedCount int
	err := r.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failedCount)
	if err != nil {
		return err
	}
	if failedCount > 0 {
		fmt.Println("Router blocked by failed task. Waiting for Owner intervention...")
		return nil
	}

	// 2. Ensure we only process one task at a time (triad of 3 agents working on one task)
	lockFile := ".build/router_working.lock"
	if _, err := os.Stat(lockFile); err == nil {
		// Currently processing a task
		return nil
	}

	// 3. Find the next actionable 'todo' leaf task
	row := r.db.QueryRow(`
		SELECT t.id, t.title, t.description, t.agent_id 
		FROM tasks t
		WHERE t.status = 'todo'
		AND NOT EXISTS (
			SELECT 1 FROM tasks c WHERE c.parent_id = t.id AND c.status = 'todo'
		)
		ORDER BY t.rowid ASC
		LIMIT 1
	`)

	var id, title string
	var description sql.NullString
	var assigneeID sql.NullInt64

	err = row.Scan(&id, &title, &description, &assigneeID)
	if err == sql.ErrNoRows {
		return nil // Nothing to do
	} else if err != nil {
		return err
	}

	// Default to dev if no assignee
	currentAssignee := 2
	if assigneeID.Valid && assigneeID.Int64 > 0 {
		currentAssignee = int(assigneeID.Int64)
	} else {
		// Update DB to reflect initial assignment
		_, _ = r.db.Exec("UPDATE tasks SET agent_id = 2 WHERE id = ?", id)
	}

	// Acquire lock
	os.WriteFile(lockFile, []byte(id), 0644)
	
	// Process the task in a goroutine so it doesn't block the next ticker tick from checking failed states,
	// but the lockfile ensures we don't start a second agent.
	go func() {
		defer os.Remove(lockFile)
		r.processTask(id, title, description.String, currentAssignee)
	}()

	return nil
}

func (r *Router) processTask(taskID, title, description string, assigneeID int) {
	fmt.Printf("\n--- Processing Task %s with Assignee %d ---\n", taskID, assigneeID)

	var roleFile string
	switch assigneeID {
	case 2:
		roleFile = "cmd/build/templates/dev.md"
	case 3:
		roleFile = "cmd/build/templates/tester.md"
	case 4:
		roleFile = "cmd/build/templates/boss.md"
	default:
		return
	}

	// Combine instructions
	agentBytes, _ := os.ReadFile(roleFile)
	contextContent := fmt.Sprintf("\n\n---\n### YOUR CURRENT ASSIGNMENT\nTask ID: %s\n\nPlease run `script/context %s` to retrieve the task description and comments history before you begin.\n", taskID, taskID)
	
	fullInstructions := string(agentBytes) + contextContent

	agentInstructionFile := fmt.Sprintf(".build/agent_%d.md", assigneeID)
	os.WriteFile(agentInstructionFile, []byte(fullInstructions), 0644)
	defer os.Remove(agentInstructionFile)

	// Run opencode CLI session
	fmt.Printf("Launching opencode for %s...\n", roleFile)
	
	cmd := exec.Command("opencode", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("opencode session ended with error: %v\n", err)
	}

	// Post-session logic
	r.handlePostSession(taskID, assigneeID)
}

func (r *Router) handlePostSession(taskID string, assigneeID int) {
	// Re-check status in case the agent (like Boss) changed it to 'done'
	var status string
	var attempts int
	r.db.QueryRow("SELECT status, approval_attempts FROM tasks WHERE id = ?", taskID).Scan(&status, &attempts)

	if status == "done" {
		fmt.Printf("Task %s is marked done.\n", taskID)
		return
	}

	switch assigneeID {
	case 2: // Dev finished -> Hand off to Tester
		r.db.Exec("UPDATE tasks SET agent_id = 3 WHERE id = ?", taskID)
	case 3: // Tester finished -> Run tests
		fmt.Println("Running test suite...")
		testCmd := exec.Command("script/test")
		out, err := testCmd.CombinedOutput()
		
		if err != nil {
			fmt.Printf("Tests failed. Kicking back to Dev.\n")
			// Add comment with output
			commentText := fmt.Sprintf("Tests failed:\n```\n%s\n```", string(out))
			r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 3, ?)", taskID, commentText)
			
			attempts++
			if attempts >= 3 {
				r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
			} else {
				r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
			}
		} else {
			fmt.Println("Tests passed. Handing off to Boss.")
			r.db.Exec("UPDATE tasks SET agent_id = 4 WHERE id = ?", taskID)
		}
	case 4: // Boss finished but task is still 'todo' (Disapproved)
		fmt.Printf("Boss exited without approving. Kicking back to Dev.\n")
		attempts++
		if attempts >= 3 {
			r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
		} else {
			r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
		}
	}
}



