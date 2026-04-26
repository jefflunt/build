package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jefflunt/build/internal/db"
	"github.com/jefflunt/build/internal/router"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: build <subcommand>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "help":
		fmt.Println("build help - show this help")
		fmt.Println("build start - start the router service")
		fmt.Println("build status - show router status")
		fmt.Println("build seed - seed the database with test data")
	case "start":
		runRouter()
	case "status":
		fmt.Println("Router status: check PID file in data/router.pid")
	case "seed":
		seedDB()
	default:
		fmt.Printf("Unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func seedDB() {
	database, err := db.InitDB("data/build.db")
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
	// Initialize DB
	database, err := db.InitDB("data/build.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Write PID
	pid := os.Getpid()
	err = os.WriteFile("data/router.pid", []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write PID: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove("data/router.pid")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start Router
	r := router.NewRouter(database)
	go r.Run()

	<-sigChan
	fmt.Println("Router stopped.")
}
