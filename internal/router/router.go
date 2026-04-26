package router

import (
	"database/sql"
	"fmt"
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
	ticker := time.NewTicker(5 * time.Second) // Adjust as needed
	for range ticker.C {
		if err := r.reconcile(); err != nil {
			fmt.Printf("Error reconciling: %v\n", err)
		}
	}
	return nil
}

// reconcile implements the core state machine logic.
func (r *Router) reconcile() error {
	fmt.Println("Reconciling state...")

	// 1. Monitor: Query actionable entities
	// We need to be careful with locking here.
	// Let's use a transaction or ensure connections are closed.
	
	rows, err := r.db.Query(`
		SELECT t.id, t.type, t.assignee_id 
		FROM tasks t
		WHERE t.status = 'todo'
		AND NOT EXISTS (
			SELECT 1 FROM tasks c WHERE c.parent_id = t.id AND c.status = 'todo'
		)
	`)
	if err != nil {
		return err
	}
	
	// Collect actionable entities FIRST, then close rows
	type Actionable struct {
		id string
		entityType string
	}
	var actionable []Actionable
	for rows.Next() {
		var id, entityType string
		var assigneeID sql.NullInt64
		if err := rows.Scan(&id, &entityType, &assigneeID); err != nil {
			rows.Close()
			return err
		}
		if !assigneeID.Valid || assigneeID.Int64 == 0 {
			actionable = append(actionable, Actionable{id, entityType})
		}
	}
	rows.Close() // Close before updating

	// 2. Assignment Engine
	for _, a := range actionable {
		// Just a placeholder until Agent registry is fully functional
		// For now we set to 1 (Owner) as a fallback
		_, err = r.db.Exec("UPDATE tasks SET assignee_id = 1 WHERE id = ?", a.id)
		if err != nil {
			fmt.Printf("DEBUG: Update error: %v\n", err)
			return err
		}
		fmt.Printf("Assigned %s to Owner\n", a.id)
	}

	// 3. Escalation Logic
	_, err = r.db.Exec("UPDATE tasks SET touch_count = touch_count + 1 WHERE status = 'todo'")
	return err
}
