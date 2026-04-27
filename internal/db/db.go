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
		agent_id INTEGER,
		touch_count INTEGER DEFAULT 0,
		approval_attempts INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT,
		name TEXT
	);
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		actor_id INTEGER,
		action TEXT,
		llm_provider TEXT,
		llm_model TEXT,
		llm_instructions_sha256 TEXT,
		build_version TEXT,
		duration_seconds INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		agent_id INTEGER,
		content TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(schema)
	return db, err
}
