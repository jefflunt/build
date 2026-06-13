# Design: CLI-Owned Database and State Operations (`build task`)

## User Story
* **Headline**: Expose Database and Workflow State Operations via `build task` Subcommands
* **Problem Statement**: The current Node-RED flow calls the raw `sqlite3` CLI directly inside JavaScript function nodes. This couples Node-RED tightly to the SQLite database file path, table structures, and syntax. It is extremely fragile to shell-quoting bugs and prone to sqlite lockup issues because operations are not properly sequentialized or managed inside the central Go engine.
* **Objective**: Introduce a unified `build task` command group in the Go `build` CLI that handles all task queries and status/assignee updates, returning structured JSON for Node-RED to consume.
* **Expected Outcome**: Node-RED becomes entirely database-blind. It interacts only with clean, safe CLI endpoints like `build task next` or `build task update`.

---

## Architecture Overview

Instead of Node-RED embedding SQL strings, it executes standard CLI subcommands on the `build` binary. The `build` CLI is responsible for loading the database, running safe Go queries, initiating atomic transactions, and returning clean JSON.

```
+------------------+                   +------------------+                   +------------------+
|                  |   build task next |                  |   SQL (Go driver) |                  |
|     Node-RED     | =================>|    build CLI     | =================>|     SQLite DB    |
| (Pure Workflow)  | <================ |  (Go execution)  | <================ |  (.build/db.db)  |
|                  |    JSON Output    |                  |    Raw results    |                  |
+------------------+                   +------------------+                   +------------------+
```

### Proposed CLI Interface

We will add the following subcommands under `build task`:

1. `build task next`:
   - Prints the next actionable `todo` task in JSON format.
   - If no task is available, prints `{}` or `null`.
2. `build task blocked`:
   - Checks if any task is in the `failed` state (blocking the workflow).
   - Prints the blocked task details as JSON, or `{}` if none.
3. `build task stuck`:
   - Checks if any task is in the `stuck` state (requiring lead escalation).
   - Prints the stuck task details as JSON, or `{}` if none.
4. `build task update --id <id> [flags]`:
   - Updates task attributes in a single transaction.
   - **Supported Flags**:
     - `--status <status>`: Updates task status (`todo`, `stuck`, `failed`, `done`).
     - `--agent-id <id>`: Updates the current assignee agent (`1` to `5`).
     - `--inc-approval-attempts`: Increments `approval_attempts` by 1.
     - `--inc-lead-interventions`: Increments `lead_interventions` by 1.
     - `--reset-approval-attempts`: Resets `approval_attempts` to 0.
     - `--comment "<text>"`: Inserts a comment associated with this task.
     - `--comment-author-id <id>`: Specifies the author ID of the inserted comment (defaults to 1).
     - `--audit-action <action>`: Inserts a row in the `audit_logs` table for the action.
     - `--duration <seconds>`: Duration field for the audit log (optional).

---

## Checklist & TDD Requirements

1. **Deterministic Test Cases**:
   - Write Go unit tests in `internal/db` or a new package/command test verifying that each subcommand behaves correctly.
   - Verify that `build task next` returns the correct task order matching current database priorities.
   - Verify that flag combinations on `build task update` are executed atomically and safely under a single database transaction.
2. **JSON Compliance**:
   - All standard output for queries (`next`, `blocked`, `stuck`) must be valid, parseable JSON.
3. **Robust Locking**:
   - Ensure the SQLite DB connection uses write-ahead logging (WAL) or proper transactional locking so that CLI commands never deadlock if invoked concurrently.

---

## Implementation Backlog

## Pending

## Current

## Completed
- Task 1: Implement the querying subcommands (`next`, `blocked`, and `stuck`) under `build task` (including CLI routing, database queries, and unit tests).
- Task 2: Implement the `build task update` subcommand (including flag parsing, atomic multi-table transactions, comments/audit creation, and unit tests).
