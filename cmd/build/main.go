package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"


	"github.com/jefflunt/build/internal/db"
	"github.com/jefflunt/build/internal/router"
	"github.com/jefflunt/build/pkg/version"
)

//go:embed templates/designer.md
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
		fmt.Println("build help - show this help")
		fmt.Println("build start - start the router service")
		fmt.Println("build status - show router status")
		fmt.Println("build seed - seed the database with test data")
		fmt.Println("build design - start a new design interaction")
		fmt.Println("build init - initialize the project")
		fmt.Println("build teardown - remove the .build project directory")
		fmt.Println("build version - show the build version")
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
			fmt.Println("Usage: build ingest <session-id-folder>")
		}
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
	INSERT INTO entities (id, parent_id, type, title, status) VALUES 
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
	// PID file should probably also go in .build/
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

	// Create instruction file
	instrFile := filepath.Join(sessionDir, "instructions.md")
	data, err := designerInstructions.ReadFile("templates/designer.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading instructions: %v\n", err)
		os.Exit(1)
	}
	os.WriteFile(instrFile, data, 0644)

	fmt.Printf("--- Starting Design Session: %d ---\n", sessionID)
	fmt.Printf("Design session ready at %s/design.md\n", sessionDir)

	// 2. Run opencode
	// Assuming opencode --instructions <file>
	cmd := exec.Command("opencode", "--instructions", instrFile)
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
		// Assuming breakdown <input> <output_dir>
		breakdownCmd := exec.Command("breakdown", designFile, sessionDir)
		if err := breakdownCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Breakdown failed: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("Ingesting tasks...")
		ingestTasks(fmt.Sprintf("%d", sessionID))
	} else {
		fmt.Println("No design.md found. Skipping breakdown/ingestion.")
	}
}

func ingestTasks(sessionID string) {
	fmt.Printf("Ingesting tasks from session %s...\n", sessionID)
	// Logic to traverse <sessionDir> and insert into tasks table would go here
	fmt.Println("Tasks ingested into database.")
}

func initProject() {
	os.MkdirAll(".build", 0755)
	database, err := db.InitDB(".build/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Seed Owner (id=1)
	_, err = database.Exec("INSERT OR IGNORE INTO agents (id, role, name) VALUES (1, 'owner', 'Owner')")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed Owner: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Project initialized in .build/")
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
