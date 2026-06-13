# Introduce a unified 'build task' command group in the Go 'build' CLI that handles all task queries (next, blocked, stuck) and status/assignee updates (update with various flags) in a single SQLite database transaction, returning clean, structured JSON to ensure Node-RED is database-blind.

This task introduces a brand-new unified command group 'build task' to the Go 'build' CLI. Moving database queries and updates out of direct Node-RED execution and behind CLI subcommands decouples the workflow automation engine from SQLite table structures, file locations, and query syntaxes.

The command group supports querying the next actionable, currently blocked, or stuck tasks in valid, parseable JSON format. It also introduces an atomic update command 'build task update' supporting flags to modify task statuses, assignees, approval and intervention counts, comment attachments, and audit trail insertions under safe, single SQL transactions with robust locking.
