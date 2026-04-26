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
		SELECT e.id, e.type, e.assigned_agent 
		FROM entities e
		WHERE e.status = 'todo'
		AND NOT EXISTS (
			SELECT 1 FROM entities c WHERE c.parent_id = e.id AND c.status = 'todo'
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
		var assignedAgent sql.NullString
		if err := rows.Scan(&id, &entityType, &assignedAgent); err != nil {
			rows.Close()
			return err
		}
		if !assignedAgent.Valid || assignedAgent.String == "" {
			actionable = append(actionable, Actionable{id, entityType})
		}
	}
	rows.Close() // Close before updating

	// 2. Assignment Engine
	for _, a := range actionable {
		newAgent := r.mapEntityToRole(a.entityType)
		_, err = r.db.Exec("UPDATE entities SET assigned_agent = ? WHERE id = ?", newAgent, a.id)
		if err != nil {
			fmt.Printf("DEBUG: Update error: %v\n", err)
			return err
		}
		fmt.Printf("Assigned %s to %s\n", a.id, newAgent)
	}

	// 3. Escalation Logic
	_, err = r.db.Exec("UPDATE entities SET touch_count = touch_count + 1 WHERE status = 'todo'")
	return err
}

func (r *Router) mapEntityToRole(entityType string) string {
	switch entityType {
	case "task":
		return "Dev"
	case "issue":
		return "Lead"
	case "epic":
		return "Boss"
	default:
		return "Deputy"
	}
}
