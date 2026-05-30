package router

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
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
	templatesFS      embed.FS
}

// NewRouter creates a new router instance.
func NewRouter(db *sql.DB, provider, model string, templatesFS embed.FS) *Router {
	return &Router{db: db, provider: provider, model: model, templatesFS: templatesFS}
}

// Run starts the persistent reconciliation loop.
func (r *Router) Run() error {
	fmt.Printf("Router service started (provider: %s, model: %s)...\n", r.provider, r.model)
	for {
		if err := r.reconcile(); err != nil {
			fmt.Printf("Error reconciling: %v\n", err)
			time.Sleep(1 * time.Second) // Prevent busy loop on error
		} else {
			// Small delay to prevent 100% CPU usage when idle or polling
			time.Sleep(1 * time.Second)
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

	// 1.5. Check if blocked by ANY stuck task
	var stuckID string
	err = r.db.QueryRow("SELECT id FROM tasks WHERE status = 'stuck' LIMIT 1").Scan(&stuckID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if stuckID != "" {
		currentState := "stuck:" + stuckID
		if r.lastPrintedState != currentState {
			r.printTree(stuckID, 5, "")
			fmt.Println("\nRouter delegating stalled task to Lead Engineer...")
			r.lastPrintedState = currentState
		}
		
		// Fetch info for the stuck task
		var title string
		var description sql.NullString
		err = r.db.QueryRow("SELECT title, description FROM tasks WHERE id = ?", stuckID).Scan(&title, &description)
		if err != nil {
			return err
		}
		
		// Process the stuck task as the Lead (Assignee 5)
		r.processTask(stuckID, title, description.String, 5)
		return nil
	}

	// 2. Find the next actionable 'todo' leaf task
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

	// Process the task synchronously
	r.processTask(id, title, description.String, currentAssignee)

	return nil
}

func (r *Router) processTask(taskID, title, description string, assigneeID int) {
	fmt.Printf("\n--- Processing Task %s with Assignee %d ---\n", taskID, assigneeID)

	var roleFile string
	var agentName string
	switch assigneeID {
	case 2:
		roleFile = "templates/dev.md"
		agentName = "build"
	case 3:
		roleFile = "templates/tester.md"
		agentName = "build"
	case 4:
		roleFile = "templates/boss.md"
		agentName = "build"
	case 5:
		roleFile = "templates/lead.md"
		agentName = "build"
	default:
		return
	}

	// Fetch comments history
	rows, err := r.db.Query("SELECT a.role, c.content FROM comments c JOIN agents a ON c.agent_id = a.id WHERE c.task_id = ? ORDER BY c.id ASC", taskID)
	var commentsHistory string
	if err == nil {
		for rows.Next() {
			var role, content string
			rows.Scan(&role, &content)
			commentsHistory += fmt.Sprintf("%s: %s\n----------------------------------------\n", role, content)
		}
		rows.Close()
	}

	if commentsHistory == "" {
		commentsHistory = "(No comments yet)"
	}

	// Combine instructions
	agentBytes, err := r.templatesFS.ReadFile(roleFile)
	if err != nil {
		fmt.Printf("Error reading template %s: %v\n", roleFile, err)
		return
	}
	contextContent := fmt.Sprintf("\n\n---\n### YOUR CURRENT ASSIGNMENT\nTask ID: %s\nTitle: %s\nDescription: %s\n\n### COMMENTS HISTORY\n%s\n", taskID, title, description, commentsHistory)
	
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
	runErr := cmd.Run()
	duration := int(time.Since(startTime).Seconds())
	if runErr != nil {
		fmt.Printf("opencode session ended with error: %v\n", runErr)
	}

	// Run sweep and commit before handoff
	roleName := "Unknown"
	switch assigneeID {
	case 2: roleName = "Dev"
	case 3: roleName = "Tester"
	case 4: roleName = "Boss"
	case 5: roleName = "Lead"
	}
	
	// Update tree to show sweep is running
	currentState := fmt.Sprintf("active:%s:6", taskID)
	if r.lastPrintedState != currentState {
		r.printTree(taskID, 6, "")
		r.lastPrintedState = currentState
	}
	
	r.runSweepAndCommit(taskID, roleName)

	// Post-session logic
	r.handlePostSession(taskID, assigneeID, sha256Str, duration, agentName)
}

func (r *Router) handlePostSession(taskID string, assigneeID int, instructionsSHA256 string, agentDuration int, agentName string) {
	// Re-check status in case the agent (like Boss) changed it to 'done'
	var status string
	var attempts, leadInterventions int
	err := r.db.QueryRow("SELECT status, approval_attempts, lead_interventions FROM tasks WHERE id = ?", taskID).Scan(&status, &attempts, &leadInterventions)
	if err != nil {
		fmt.Printf("Error checking task status: %v\n", err)
	}

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
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		testCmd := exec.CommandContext(ctx, "./.build/test")
		out, err := testCmd.CombinedOutput()
		testDuration := int(time.Since(testStartTime).Seconds())
		totalDuration := agentDuration + testDuration
		
		if err != nil {
			fmt.Printf("Tests failed. Kicking back to Dev.\n")
			var commentText string
			if ctx.Err() == context.DeadlineExceeded {
				commentText = fmt.Sprintf("Tests failed: Execution timed out after 30 seconds. Your tests are running too slowly. This usually means you are making real network requests, hitting a real database, or have inefficient loops. Please refactor the codebase to isolate I/O and use mocks/stubs in your tests.\n\nPartial output:\n```\n%s\n```", string(out))
			} else {
				commentText = fmt.Sprintf("Tests failed:\n```\n%s\n```", string(out))
			}
			r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 3, ?)", taskID, commentText)
			
			attempts++
			if attempts >= 3 {
				if leadInterventions < 3 {
					// Escalate to Lead
					r.db.Exec("UPDATE tasks SET status = 'stuck', agent_id = 5, approval_attempts = 0, lead_interventions = lead_interventions + 1 WHERE id = ?", taskID)
					r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'escalate_to_lead', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
					r.lastPrintedState = "stuck:" + taskID
					r.printTree(taskID, 5, "")
				} else {
					// Hard failure
					r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
					r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_rejected', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
					r.lastPrintedState = "failed:" + taskID
					r.printTree("", 0, taskID)
				}
			} else {
				r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
				r.lastPrintedState = fmt.Sprintf("active:%s:2", taskID)
				r.printTree(taskID, 2, "")
			}
		} else {
			fmt.Println("Tests passed. Handing off to Boss.")
			
			// Concise instruction for the Boss
			instructionMsg := "**CRITICAL**: Please review task " + taskID + " and provide your feedback using the `build review` command.\n\n" +
				"The syntax is:\n" +
				"build review " + taskID + " <approve|reject> \"<reasoning>\"\n\n" +
				"RULES:\n" +
				"1. Your reasoning MUST be a single line of text.\n" +
				"2. Enclose your reasoning in double quotes (\").\n" +
				"3. DO NOT use double quotes inside your reasoning string (use single quotes instead).\n" +
				"4. DO NOT ask questions; provide ONLY the final review via the bash tool."
			r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 1, ?)", taskID, instructionMsg)

			r.db.Exec("UPDATE tasks SET agent_id = 4 WHERE id = ?", taskID)
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_boss', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, totalDuration, agentName)
			r.lastPrintedState = fmt.Sprintf("active:%s:4", taskID)
			r.printTree(taskID, 4, "")
		}
	case 4: // Boss finished
		var commentContent string
		errComment := r.db.QueryRow("SELECT content FROM comments WHERE task_id = ? AND agent_id = 4 ORDER BY id DESC LIMIT 1", taskID).Scan(&commentContent)
		if errComment != nil {
			r.kickBackToBoss(taskID, "System Error: You exited without leaving a review. You MUST use `build review` to leave your evaluation before exiting.", instructionsSHA256, attempts)
			return
		}

		// Clean potential markdown codeblocks out of the comment before parsing
		cleanedComment := strings.ReplaceAll(commentContent, "```json", "")
		cleanedComment = strings.ReplaceAll(cleanedComment, "```", "")
		cleanedComment = strings.TrimSpace(cleanedComment)

		// 2. Parse JSON
		var payload map[string]interface{}
		errJson := json.Unmarshal([]byte(cleanedComment), &payload)
		if errJson != nil {
			r.kickBackToBoss(taskID, "System Error: Your review was not correctly formatted. Please use the `build review` command.", instructionsSHA256, attempts)
			return
		}

		// 3. Strict schema validation
		if len(payload) != 2 {
			r.kickBackToBoss(taskID, "System Error: Your review payload must contain exactly two keys: 'reasoning' and 'approval'. Please use the `build review` command.", instructionsSHA256, attempts)
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
				if leadInterventions < 3 {
					// Escalate to Lead
					r.db.Exec("UPDATE tasks SET status = 'stuck', agent_id = 5, approval_attempts = 0, lead_interventions = lead_interventions + 1 WHERE id = ?", taskID)
					r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'escalate_to_lead', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
					r.lastPrintedState = "stuck:" + taskID
					r.printTree(taskID, 5, "")
				} else {
					// Hard failure
					r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
					r.lastPrintedState = "failed:" + taskID
					r.printTree("", 0, taskID)
				}
			} else {
				r.db.Exec("UPDATE tasks SET agent_id = 2, approval_attempts = ? WHERE id = ?", attempts, taskID)
				r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
				r.lastPrintedState = fmt.Sprintf("active:%s:2", taskID)
				r.printTree(taskID, 2, "")
			}
		}
	case 5: // Lead finished
		// Check if the Lead left a comment
		var commentCount int
		r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE task_id = ? AND agent_id = 5", taskID).Scan(&commentCount)
		
		if commentCount == 0 {
			// Lead failed to leave a comment, but we have to push it back to the dev anyway so it doesn't get stuck forever
			// We'll leave an automated comment
			r.db.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, 1, ?)", taskID, "System Note: The Lead Engineer reviewed the task but failed to leave explicit instructions. Developer, please carefully review the previous failures and try again.")
		}

		// Always push back to Dev after Lead intervention
		r.db.Exec("UPDATE tasks SET status = 'todo', agent_id = 2 WHERE id = ?", taskID)
		r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'assign_to_dev', ?, ?, ?, ?, ?, ?)", taskID, r.provider, r.model, instructionsSHA256, buildVersion, agentDuration, agentName)
		r.lastPrintedState = fmt.Sprintf("active:%s:2", taskID)
		r.printTree(taskID, 2, "")
	}
}

func (r *Router) kickBackToBoss(taskID string, errMsg string, instructionsSHA256 string, attempts int) {
	fmt.Printf("Boss validation failed format check. Kicking back to Boss.\n")
	
	attempts++

	var leadInterventions int
	r.db.QueryRow("SELECT lead_interventions FROM tasks WHERE id = ?", taskID).Scan(&leadInterventions)

	if attempts >= 3 {
		if leadInterventions < 3 {
			// Escalate to Lead
			r.db.Exec("UPDATE tasks SET status = 'stuck', agent_id = 5, approval_attempts = 0, lead_interventions = lead_interventions + 1 WHERE id = ?", taskID)
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'escalate_to_lead', ?, ?, ?, ?, 0, 'plan')", taskID, r.provider, r.model, instructionsSHA256, version.Version)
			r.lastPrintedState = "stuck:" + taskID
			r.printTree(taskID, 5, "")
		} else {
			// Hard failure
			r.db.Exec("UPDATE tasks SET status = 'failed', agent_id = 1, approval_attempts = ? WHERE id = ?", attempts, taskID)
			r.db.Exec("INSERT INTO audit_logs (task_id, actor_id, action, llm_provider, llm_model, llm_instructions_sha256, build_version, duration_seconds, opencode_agent) VALUES (?, 1, 'task_failed_max_attempts', ?, ?, ?, ?, 0, 'plan')", taskID, r.provider, r.model, instructionsSHA256, version.Version)
			r.lastPrintedState = "failed:" + taskID
			r.printTree("", 0, taskID)
		}
		return
	}

	fullMsg := errMsg + `

CRITICAL REMINDER: You must use your bash/shell tool to execute the 'build review' command.

The syntax is:
build review ` + taskID + ` <approve|reject> "<reasoning>"

RULES:
1. Your reasoning MUST be a single line of text.
2. Enclose your reasoning in double quotes (").
3. DO NOT use double quotes inside your reasoning string (use single quotes instead).

Example:
build review ` + taskID + ` approve "The code looks good and tests pass."`

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
		} else if status == "stuck" {
			prefix = "\033[96m" // Light Cyan
		} else if id == activeID {
			switch activeAssignee {
			case 2:
				prefix = "\033[94m" // Light Blue
			case 3:
				prefix = "\033[93m" // Light Yellow
			case 4:
				prefix = "\033[92m" // Light Green
			case 5:
				prefix = "\033[96m" // Light Cyan
			case 6:
				prefix = "\033[30;47m" // Black text on White background
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
func (r *Router) runSweepAndCommit(taskID, role string) {
	fmt.Println("\n--- Running 'sweep' agent to update .gitignore ---")
	
	// Initialize git repo if it doesn't exist
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("Initializing git repository...")
		exec.Command("git", "init").Run()
	}

	agentBytes, err := r.templatesFS.ReadFile("templates/sweep.md")
	if err != nil {
		fmt.Printf("Error reading sweep template: %v\n", err)
		return
	}
	
	sweepInstructionsFile := fmt.Sprintf(".build/sweep_%s.md", taskID)
	os.WriteFile(sweepInstructionsFile, agentBytes, 0644)
	defer os.Remove(sweepInstructionsFile)
	
	cmd := exec.Command("opencode", "-m", fmt.Sprintf("%s/%s", r.provider, r.model), "--agent", "build", "run", string(agentBytes))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Sweep agent ended with error: %v\n", err)
	}
	
	fmt.Println("Committing changes...")
	exec.Command("git", "add", ".").Run()
	
	statusCmd := exec.Command("git", "status", "--porcelain")
	out, _ := statusCmd.Output()
	if len(strings.TrimSpace(string(out))) > 0 {
		commitMsg := fmt.Sprintf("build: updates for task %s by %s", taskID, role)
		commitCmd := exec.Command("git", "commit", "-m", commitMsg)
		commitCmd.Stdout = os.Stdout
		commitCmd.Stderr = os.Stderr
		commitCmd.Run()
		fmt.Println("Changes committed successfully.")
	} else {
		fmt.Println("No changes to commit.")
	}
}
