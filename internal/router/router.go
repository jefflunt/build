package router

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jefflunt/build/pkg/version"
)

// Router represents the persistent background service.
type Router struct {
	db               *sql.DB
	lastPrintedState string
	provider         string
	model            string
}

// NewRouter creates a new router instance.
func NewRouter(db *sql.DB, provider, model string) *Router {
	return &Router{db: db, provider: provider, model: model}
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
	var failedID string
	err := r.db.QueryRow("SELECT id FROM tasks WHERE status = 'failed' LIMIT 1").Scan(&failedID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if failedID != "" {
		currentState := "failed:" + failedID
		if r.lastPrintedState != currentState {
			r.printTree("", 0, failedID)
			fmt.Println("\nRouter blocked by failed task. Waiting for Owner intervention...")
			r.lastPrintedState = currentState
		}
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
		// Check if any todo tasks exist
		var todoCount int
		err := r.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'todo'").Scan(&todoCount)
		if err == nil && todoCount == 0 {
			if r.lastPrintedState != "all_finished" {
				fmt.Println("All current items have been finished")
				r.lastPrintedState = "all_finished"
			}
		} else if r.lastPrintedState != "idle" {
			r.lastPrintedState = "idle"
		}
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
		r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, 'build')", id, r.provider, r.model)
	}

	currentState := fmt.Sprintf("active:%s:%d", id, currentAssignee)
	if r.lastPrintedState != currentState {
		r.printTree(id, currentAssignee, "")
		r.lastPrintedState = currentState
	}

	// Acquire lock
	os.WriteFile(lockFile, []byte(id), 0644)
	
	// Process the task in a goroutine
	go func() {
		defer os.Remove(lockFile)
		r.processTask(id, title, description.String, currentAssignee)
	}()

	return nil
}

func (r *Router) processTask(taskID, title, description string, assigneeID int) {
	fmt.Printf("\n--- Processing Task %s with Assignee %d ---\n", taskID, assigneeID)

	var roleFile string
	var agentName string
	switch assigneeID {
	case 2:
		roleFile = "cmd/build/templates/dev.md"
		agentName = "build"
	case 3:
		roleFile = "cmd/build/templates/tester.md"
		agentName = "build"
	case 4:
		roleFile = "cmd/build/templates/boss.md"
		agentName = "plan"
	default:
		return
	}

	// Combine instructions
	agentBytes, _ := os.ReadFile(roleFile)
	contextContent := fmt.Sprintf("\n\n---\n### YOUR CURRENT ASSIGNMENT\nTask ID: %s\n\nPlease run `build context %s` to retrieve the task description and comments history before you begin.\n", taskID, taskID)
	
	fullInstructions := string(agentBytes) + contextContent
	
	// Calculate SHA256 of instructions
	hash := sha256.Sum256([]byte(fullInstructions))
	sha256Str := hex.EncodeToString(hash[:])

	agentInstructionFile := fmt.Sprintf(".build/agent_%d.md", assigneeID)
	os.WriteFile(agentInstructionFile, []byte(fullInstructions), 0644)
	defer os.Remove(agentInstructionFile)

	// Run autonomous opencode CLI session
	fmt.Printf("Launching autonomous opencode session for %s with model %s/%s as agent %s...\n", roleFile, r.provider, r.model, agentName)
	
	cmd := exec.Command("opencode", "-m", fmt.Sprintf("%s/%s", r.provider, r.model), "--agent", agentName, "run", fullInstructions)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	startTime := time.Now()
	err := cmd.Run()
	duration := int(time.Since(startTime).Seconds())
	if err != nil {
		fmt.Printf("opencode session ended with error: %v\n", err)
	}

	// Post-session logic
	r.handlePostSession(taskID, assigneeID, sha256Str, duration, agentName)
}

func (r *Router) handlePostSession(taskID string, assigneeID int, instructionsSHA256 string, agentDuration int, agentName string) {
	// Re-check status in case the agent (like Boss) changed it to 'done'
	var status string
	var attempts int
	r.db.QueryRow("SELECT status, approval_attempts FROM tasks WHERE id = ?", taskID).Scan(&status, &attempts)

	if status == "done" {
		fmt.Printf("Task %s is marked done.\n", taskID)
		r.lastPrintedState = "done:" + taskID
		r.printTree("", 0, "")
		return
	}

	buildVersion := version.Version

	switch assigneeID {
	case 2: // Dev finished -> Hand off to Tester
		r.db.Exec("UPDATE tasks SET agent_id = 3 WHERE id = ?", taskID)
		r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_tester', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, "build")
		r.lastPrintedState = fmt.Sprintf("active:%s:3", taskID)
		r.printTree(taskID, 3, "")
	case 3: // Tester finished -> Run tests
		fmt.Println("Running test suite...")
		testStartTime := time.Now()
		testCmd := exec.Command("./.build/test")
		out, err := testCmd.CombinedOutput()
		testDuration := int(time.Since(testStartTime).Seconds())
		totalDuration := agentDuration + testDuration
		
		if err != nil {
			fmt.Printf("Tests failed. Kicking back to Dev.\n")
			commentText := fmt.Sprintf("Tests failed:\n```\n%s\n```", string(out))
			r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 3, ?)", taskID, commentText)
			
			attempts++
			if attempts >= 3 {
				r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_rejected', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
				r.lastPrintedState = "failed:" + taskID
				r.printTree("", 0, taskID)
			} else {
				r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
				r.lastPrintedState = fmt.Sprintf("active:%s:2", taskID)
				r.printTree(taskID, 2, "")
			}
		} else {
			fmt.Println("Tests passed. Handing off to Boss.")
			
			r.db.Exec("UPDATE tasks SET agent_id = 4 WHERE id = ?", taskID)
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_boss', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
			r.lastPrintedState = fmt.Sprintf("active:%s:4", taskID)
			r.printTree(taskID, 4, "")
		}
	case 4: // Boss finished
		var commentContent string
		err := r.db.QueryRow("SELECT content FROM comments WHERE task_id = ? AND agent_id = 4 ORDER BY id DESC LIMIT 1", taskID).Scan(&commentContent)
		if err != nil {
			r.kickBackToBoss(taskID, "System Error: You exited without leaving a comment. You MUST use `build comment` to leave your JSON evaluation before exiting.", instructionsSHA256, attempts)
			return
		}

		// Clean potential markdown codeblocks out of the comment before parsing
		cleanedComment := strings.ReplaceAll(commentContent, "```json", "")
		cleanedComment = strings.ReplaceAll(cleanedComment, "```", "")
		cleanedComment = strings.TrimSpace(cleanedComment)

		// 2. Parse JSON
		var payload map[string]interface{}
		err = json.Unmarshal([]byte(cleanedComment), &payload)
		if err != nil {
			r.kickBackToBoss(taskID, "System Error: Your comment was not valid JSON. Please provide exactly the required JSON format.", instructionsSHA256, attempts)
			return
		}

		// 3. Strict schema validation
		if len(payload) != 2 {
			r.kickBackToBoss(taskID, "System Error: Your JSON payload must contain exactly two keys: 'reasoning' and 'approval'.", instructionsSHA256, attempts)
			return
		}

		reasoningVal, hasReasoning := payload["reasoning"]
		approvalVal, hasApproval := payload["approval"]

		if !hasReasoning || !hasApproval {
			r.kickBackToBoss(taskID, "System Error: Missing required keys. You must provide both 'reasoning' and 'approval'.", instructionsSHA256, attempts)
			return
		}

		reasoningStr, isString := reasoningVal.(string)
		if !isString || strings.TrimSpace(reasoningStr) == "" {
			r.kickBackToBoss(taskID, "System Error: The 'reasoning' key must be a non-empty string.", instructionsSHA256, attempts)
			return
		}

		approvalBool, isBool := approvalVal.(bool)
		if !isBool {
			r.kickBackToBoss(taskID, "System Error: The 'approval' key must be a boolean (true or false).", instructionsSHA256, attempts)
			return
		}

		if approvalBool {
			fmt.Printf("Boss approved task %s.\n", taskID)
			r.db.Exec("UPDATE tasks SET status = 'done' WHERE id = ?", taskID)
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_approved', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
			r.lastPrintedState = "done:" + taskID
			r.printTree("", 0, "")
		} else {
			fmt.Printf("Boss rejected task %s. Kicking back to Dev.\n", taskID)
			attempts++
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_rejected', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
			if attempts >= 3 {
				r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.lastPrintedState = "failed:" + taskID
				r.printTree("", 0, taskID)
			} else {
				r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
				r.lastPrintedState = fmt.Sprintf("active:%s:2", taskID)
				r.printTree(taskID, 2, "")
			}
		}
	}
}

func (r *Router) kickBackToBoss(taskID string, errMsg string, instructionsSHA256 string, attempts int) {
	fmt.Printf("Boss validation failed format check. Kicking back to Boss.\n")
	
	attempts++

	if attempts >= 3 {
		r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
		r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_failed_max_attempts', ?, ?, ?, ?, 0, 'plan')", taskID, r.provider, r.model, instructionsSHA256, version.Version)
		r.lastPrintedState = "failed:" + taskID
		r.printTree("", 0, taskID)
		return
	}

	fullMsg := errMsg + `

CRITICAL REMINDER: Your comment MUST be a valid JSON object. You must use your bash/shell tool to execute exactly:
build comment <task-id> '<json>'

The JSON payload must be strictly formatted with exactly two keys:
` + "```json\n{\n  \"reasoning\": \"Detailed explanation of your evaluation...\",\n  \"approval\": boolean\n}\n```" + `
- "reasoning": A non-empty string explaining your evaluation in detail.
- "approval": A boolean (true or false). true if you approve, false if you reject.`

	r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 1, ?)", taskID, fullMsg)
	r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'boss_signoff_failure', ?, ?, ?, ?, 0, 'plan')", taskID, r.provider, r.model, instructionsSHA256, version.Version)
	r.db.Exec("UPDATE tasks SET agent_id = 4, approval_attempts = ? WHERE id = ?", attempts, taskID)
	r.lastPrintedState = fmt.Sprintf("active:%s:4", taskID)
	r.printTree(taskID, 4, "")
}

func (r *Router) printTree(activeID string, activeAssignee int, failedID string) {
	query := `
	WITH RECURSIVE task_tree AS (
		SELECT id, parent_id, title, type, status, 0 AS depth, CAST(rowid AS TEXT) AS sort_path, rowid
		FROM tasks
		WHERE parent_id = '' OR parent_id IS NULL
		UNION ALL
		SELECT t.id, t.parent_id, t.title, t.type, t.status, tt.depth + 1, tt.sort_path || '/' || substr('0000000000' || t.rowid, -10, 10), t.rowid
		FROM tasks t
		JOIN task_tree tt ON t.parent_id = tt.id
	)
	SELECT id, title, status, depth
	FROM task_tree
	ORDER BY sort_path ASC;
	`
	rows, err := r.db.Query(query)
	if err != nil {
		fmt.Printf("Error querying tree: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n======================== TASK TREE ========================")
	for rows.Next() {
		var id, title, status string
		var depth int
		if err := rows.Scan(&id, &title, &status, &depth); err != nil {
			continue
		}

		prefix := ""
		suffix := ""

		if status == "done" {
			prefix = "\033[90m" // Medium Grey
		} else if failedID != "" && id == failedID {
			prefix = "\033[91m" // Light Red
		} else if id == activeID {
			switch activeAssignee {
			case 2:
				prefix = "\033[94m" // Light Blue
			case 3:
				prefix = "\033[93m" // Light Yellow
			case 4:
				prefix = "\033[92m" // Light Green
			}
		}

		if prefix != "" {
			suffix = "\033[0m"
		}

		indent := strings.Repeat("    ", depth)
		fmt.Printf("%s%s- %s: %s (%s)%s\n", prefix, indent, id, title, status, suffix)
	}
	fmt.Println("===========================================================")
}
