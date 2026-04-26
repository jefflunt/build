package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		parent_id TEXT,
		type TEXT,
		title TEXT,
		description TEXT,
		status TEXT DEFAULT 'todo',
		assignee_id INTEGER,
		creator_id INTEGER,
		touch_count INTEGER DEFAULT 0,
		escalation_level INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT,
		name TEXT
	);
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		actor_id INTEGER,
		action TEXT,
		content TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(schema)
	return db, err
}
