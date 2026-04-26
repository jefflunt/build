package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jefflunt/build/internal/db"
	"github.com/jefflunt/build/internal/router"
	"github.com/jefflunt/build/pkg/version"
)

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
		fmt.Println("build new boss - start a new boss interaction")
		fmt.Println("build init - initialize the project")
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
	case "new":
		if len(os.Args) >= 3 && os.Args[2] == "boss" {
			startNewBoss()
		} else {
			fmt.Println("Usage: build new boss")
		}
	case "init":
		initProject()
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

func startNewBoss() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- New Boss Interaction: Goal Exploration ---")
	fmt.Println("I am your Boss. Please describe the goal or problem you want to solve.")
	fmt.Print("> ")
	goal, _ := reader.ReadString('\n')
	goal = strings.TrimSpace(goal)

	// Simulate "Grill Me"
	fmt.Println("\n[Boss] Priming 'Grill Me' protocol...")
	fmt.Println("[Boss] I understand the goal. Let's vet this.")
	fmt.Println("[Boss] Challenge: Why is this goal critical right now? What happens if we don't build it?")
	fmt.Print("> ")
	reader.ReadString('\n') // Consume human input

	fmt.Println("\n[Boss] Vetted. Now running 'breakdown' to structure the work...")
	// Logic to call breakdown and enqueue tasks would go here
	fmt.Println("[Boss] Breakdown complete. Tasks enqueued in the Router.")
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
