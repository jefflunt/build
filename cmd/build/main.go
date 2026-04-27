package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jefflunt/build/internal/db"
	"github.com/jefflunt/build/internal/router"
	"github.com/jefflunt/build/pkg/version"
)

type Node struct {
	ID          string  `json:"id"`
	ParentID    string  `json:"parent_id,omitempty"`
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Status      string  `json:"status"`
	Children    []*Node `json:"children,omitempty"`
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}


//go:embed templates/build-designer.md
var designerInstructions embed.FS

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: build <subcommand>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(version.Version)
		os.Exit(0)
	case "help":
		fmt.Println("Usage: build <subcommand>")
		fmt.Println()
		fmt.Println("Commands:")
		commands := []struct {
			name string
			desc string
		}{
			{"help", "show this help"},
			{"start", "start the router service"},
			{"status", "show router status"},
			{"seed", "seed the database with test data"},
			{"design", "start a new design interaction"},
			{"ingest", "ingest breakdown output"},
			{"context", "view task context and comments"},
			{"comment", "add a comment to a task"},
			{"approve", "approve a task (boss only)"},
			{"try_again", "reset a failed task for the dev"},
			{"init", "initialize the project"},
			{"teardown", "remove the .build project directory"},
			{"version", "show the build version"},
		}
		for _, c := range commands {
			fmt.Printf("  \x1b[33m%-15s\x1b[0m %s\n", c.name, c.desc)
		}
	case "start":
		runRouter()
	case "status":
		if _, err := os.Stat(".build/router.pid"); os.IsNotExist(err) {
			fmt.Println("Router status: stopped")
		} else {
			fmt.Println("Router status: running")
		}
	case "seed":
		seedDB()
	case "design":
		startDesigner()
	case "ingest":
		if len(os.Args) >= 3 {
			ingestTasks(os.Args[2])
		} else {
			fmt.Println("Usage: build ingest <path/to/breakdown/output/directory>")
		}
	case "context":
		if len(os.Args) < 3 {
			fmt.Println("Usage: build context <task-id>")
			os.Exit(1)
		}
		printContext(os.Args[2])
	case "comment":
		if len(os.Args) < 4 {
			fmt.Println("Usage: build comment <task-id> <comment text...>")
			os.Exit(1)
		}
		addComment(os.Args[2], strings.Join(os.Args[3:], " "))
	case "approve":
		if len(os.Args) < 3 {
			fmt.Println("Usage: build approve <task-id> [comments...]")
			os.Exit(1)
		}
		comments := ""
		if len(os.Args) > 3 {
			comments = strings.Join(os.Args[3:], " ")
		}
		approveTask(os.Args[2], comments)
	case "try_again":
		if len(os.Args) < 3 {
			fmt.Println("Usage: build try_again <task-id>")
			os.Exit(1)
		}
		tryAgain(os.Args[2])
	case "init":
		initProject()
	case "teardown":
		teardownProject()
	default:
		fmt.Printf("Unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func seedDB() {
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	seed := `
	INSERT INTO tasks (id, parent_id, type, title, status) VALUES 
	('G1', NULL, 'goal', 'Build the Orchestrator', 'todo'),
	('E1', 'G1', 'epic', 'Implement Router', 'todo'),
	('I1', 'E1', 'issue', 'Create Database Schema', 'todo'),
	('T1', 'I1', 'task', 'Define Tables', 'todo');
	`
	_, err = database.Exec(seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed DB: %v\n", err)
	} else {
		fmt.Println("Database seeded successfully.")
	}
}

func runRouter() {
	// Check if initialized
	if _, err := os.Stat(".build"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Project not initialized. Run 'build init' first.\n")
		os.Exit(1)
	}

	// Initialize DB
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Write PID
	pidDir := ".build"
	pid := os.Getpid()
	err = os.WriteFile(pidDir+"/router.pid", []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write PID: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(pidDir+"/router.pid")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start Router
	r := router.NewRouter(database)
	go r.Run()

	<-sigChan
	fmt.Println("Router stopped.")
}

func startDesigner() {
	// 1. Check for opencode
	if _, err := exec.LookPath("opencode"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: 'opencode' not found in PATH. Please install it first.\n")
		os.Exit(1)
	}

	sessionID := time.Now().UnixNano() / 1000 // Microseconds UTC
	sessionDir := fmt.Sprintf(".build/designs/%d", sessionID)
	os.MkdirAll(sessionDir, 0755)

	fmt.Printf("--- Starting Design Session: %d ---\n", sessionID)
	fmt.Printf("Design session ready at %s/design.md\n", sessionDir)

	// 2. Run opencode
	// Launch the interactive TUI.
	// The agent 'build-designer' is pre-configured in .opencode/agents/build-designer.md
	// and will be available for selection in the TUI session.
	cmd := exec.Command("opencode", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Designer session exited with error: %v\n", err)
		os.Exit(1)
	}




	// 3. Post-session: breakdown + ingest
	designFile := filepath.Join(sessionDir, "design.md")
	if _, err := os.Stat(designFile); err == nil {
		fmt.Println("Design detected. Running breakdown...")
		breakdownCmd := exec.Command("breakdown", designFile, sessionDir)
		if err := breakdownCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Breakdown failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("Ingesting tasks...")
		// Use the output directory that breakdown generates. By default, breakdown places 
		// its file structure in the target folder we gave it (`sessionDir`).
		ingestTasks(sessionDir)
	} else {
		fmt.Println("No design.md found. Skipping breakdown/ingestion.")
	}
}

func ingestTasks(targetPath string) {
	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat target path %s: %v\n", targetPath, err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: ingest requires a directory generated by the breakdown tool.\n")
		os.Exit(1)
	}

	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	rootNode, err := walkBreakdownDir(targetPath, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse breakdown directory: %v\n", err)
		os.Exit(1)
	}

	if rootNode != nil {
		insertNode(database, rootNode)
	}

	fmt.Printf("Tasks from %s ingested into database.\n", targetPath)
}

func walkBreakdownDir(dirPath string, parentID string) (*Node, error) {
	node := &Node{
		ID:       generateID(),
		ParentID: parentID,
		Status:   "todo",
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// 1. Process README.md first to get the details of this composite node
	readmePath := filepath.Join(dirPath, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		title, desc, err := parseMarkdown(readmePath)
		if err != nil {
			return nil, err
		}
		node.Title = title
		node.Description = desc
		node.Type = "composite"
	} else {
		// Fallback if no README
		node.Title = filepath.Base(dirPath)
		node.Type = "composite"
	}

	// 2. Process children
	for _, entry := range entries {
		if entry.Name() == "README.md" {
			continue
		}

		childPath := filepath.Join(dirPath, entry.Name())
		
		if entry.IsDir() {
			childNode, err := walkBreakdownDir(childPath, node.ID)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, childNode)
		} else if strings.HasSuffix(entry.Name(), ".md") {
			title, desc, err := parseMarkdown(childPath)
			if err != nil {
				return nil, err
			}
			childNode := &Node{
				ID:          generateID(),
				ParentID:    node.ID,
				Type:        "atomic",
				Title:       title,
				Description: desc,
				Status:      "todo",
			}
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}

func parseMarkdown(path string) (title, description string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		title = strings.TrimPrefix(lines[0], "# ")
		description = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	} else {
		title = filepath.Base(path)
		description = string(data)
	}
	return title, description, nil
}

func insertNode(db *sql.DB, n *Node) {
	query := `INSERT OR REPLACE INTO tasks (id, parent_id, type, title, description, status) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, n.ID, n.ParentID, n.Type, n.Title, n.Description, n.Status)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to insert node %s: %v\n", n.ID, err)
	}

	for _, child := range n.Children {
		insertNode(db, child)
	}
}

func initProject() {
	os.MkdirAll(".build", 0755)
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Seed initial agents: owner, dev, tester, boss
	_, err = database.Exec(`
		INSERT OR IGNORE INTO agents (id, role, name) VALUES 
		(1, 'owner', 'Owner'),
		(2, 'dev', 'Developer'),
		(3, 'tester', 'Tester'),
		(4, 'boss', 'Boss')
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed agents: %v\n", err)
		os.Exit(1)
	}

	// Setup designer agent in ~/.config/opencode/agents/
	home, _ := os.UserHomeDir()
	agentDir := filepath.Join(home, ".config/opencode/agents")
	os.MkdirAll(agentDir, 0755)
	agentFile := filepath.Join(agentDir, "build-designer.md")

	if _, err := os.Stat(agentFile); err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Agent 'build-designer' already exists at %s. Overwrite? [y/N]: ", agentFile)
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(response)) != "y" {
			fmt.Println("Skipping agent installation.")
		} else {
			writeAgentFile(agentFile)
		}
	} else {
		writeAgentFile(agentFile)
	}

	fmt.Println("Project initialized in .build/")
}

func writeAgentFile(path string) {
	data, err := designerInstructions.ReadFile("templates/build-designer.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading instructions: %v\n", err)
		os.Exit(1)
	}
	os.WriteFile(path, data, 0644)
	fmt.Printf("Agent 'build-designer' installed to %s\n", path)
}

func printContext(taskID string) {
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: .build/build.db not found or failed to init: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	fmt.Println("========================================")
	fmt.Printf("TASK CONTEXT: %s\n", taskID)
	fmt.Println("========================================")

	var title, desc string
	err = database.QueryRow("SELECT title, description FROM tasks WHERE id = ?", taskID).Scan(&title, &desc)
	if err == nil {
		fmt.Printf("      Title = %s\n", title)
		fmt.Printf("Description = %s\n", desc)
	}

	fmt.Println("\n========================================")
	fmt.Println("COMMENTS HISTORY")
	fmt.Println("========================================")

	rows, err := database.Query("SELECT a.role, c.content FROM comments c JOIN agents a ON c.agent_id = a.id WHERE c.task_id = ? ORDER BY c.id ASC", taskID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var role, content string
			rows.Scan(&role, &content)
			fmt.Printf("%s: %s\n----------------------------------------\n", role, content)
		}
	}
}

func addComment(taskID, comment string) {
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: .build/build.db not found or failed to init: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	var agentID sql.NullInt64
	err = database.QueryRow("SELECT agent_id FROM tasks WHERE id = ?", taskID).Scan(&agentID)
	
	// Fallback to Owner (1)
	assignee := 1
	if err == nil && agentID.Valid {
		assignee = int(agentID.Int64)
	}

	_, err = database.Exec("INSERT INTO comments (task_id, agent_id, content) VALUES (?, ?, ?)", taskID, assignee, comment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding comment: %v\n", err)
	} else {
		fmt.Printf("Comment added to task %s.\n", taskID)
	}
}

func approveTask(taskID, comments string) {
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: .build/build.db not found or failed to init: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	_, err = database.Exec("UPDATE tasks SET status='done' WHERE id = ?", taskID)
	if err == nil && comments != "" {
		database.Exec("INSERT INTO audit_log (task_id, actor_id, action, content) VALUES (?, 4, 'approve', ?)", taskID, comments)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error approving task: %v\n", err)
	} else {
		fmt.Printf("Task %s approved and marked as done.\n", taskID)
	}
}

func tryAgain(taskID string) {
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: .build/build.db not found or failed to init: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	_, err = database.Exec("UPDATE tasks SET status='todo', agent_id=2, approval_attempts=0 WHERE id = ?", taskID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating task: %v\n", err)
	} else {
		fmt.Printf("Task %s has been reset and kicked back to the Developer.\n", taskID)
	}
}

func teardownProject() {
	if _, err := os.Stat(".build"); os.IsNotExist(err) {
		fmt.Println("Project not initialized. Nothing to tear down.")
		return
	}
	
	err := os.RemoveAll(".build")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to remove .build directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Project torn down. .build/ directory removed.")
}
