# 01-Terminology

This document defines the core entities and architectural concepts for the `build` orchestrator.

## 1. The Human & Orchestrator
- **Owner**: Human (id=1), ultimate decision maker/instruction giver.
- **Router**: Deterministic background service; the *only* entity authorized to spawn agent sessions, assign agents, trigger escalations, or run tests.

## 2. The Core Agents
- **Dev** (id=2): Implements the task according to its instructions.
- **Tester** (id=3): Verifies the Dev's work by writing test cases.
- **Boss** (id=4): Evaluates the completed implementation and test suite output to verify it fulfills the entire original intent of the task.
- **Lead Engineer** (id=5): Unsticks deadlocked tasks by providing detailed code-level or architectural instructions when agents get stuck in loops.
- **Git Cleanup Artist / Sweep** (id=6): Keeps `.gitignore` up-to-date and prevents unwanted files from being committed.

## 3. Operations
- **Router Process**: Hosted inside Node-RED as an on-demand, event-driven orchestrator, triggered instantly on task changes (e.g. starting, ingesting, or redoing tasks) via HTTP POST. Also implemented as a Go-based service (`internal/router`) used in test environments.
- **CLI Commands**:
    - `build help`: Display help and descriptions for all subcommands.
    - `build init`: Initialize local `.build/` repo state, seeds agent database, writes the `build-designer.md` agent file to `~/.config/opencode/agents/`, and sets up the default `.build/test` adapter script.
    - `build start`: Writes a local PID and fires an HTTP POST notification to trigger the Node-RED orchestrator on-demand, blocking until stopped.
    - `build status`: Check if the `Router` service is running.
    - `build seed`: Seed the local SQLite database with initial task templates.
    - `build ingest <path>`: Import external `breakdown` output directory into the system database.
    - `build context <task-id>`: Retrieve and print task details, comment history, and audit history.
    - `build why-failed <task-id>`: Retrieve and print full context and audit history for a failed/stalled task.
    - `build comment <task-id> "<comment-text>"`: Append a comment to a task.
    - `build review <task-id> <approve|reject> "<reasoning>"`: Submit a structured review comment on a task with approval/rejection decision and reasoning (used by the Boss).
    - `build approve <task-id>`: Manually mark a task as `done`.
    - `build redo <task-id>`: Reset a `failed` task to `todo`, clear `approval_attempts`, and set assignee back to Dev.
    - `build teardown`: Remove the local `.build/` project directory completely.
    - `build enqueue <plan-file>`: Ingest and enqueue a plan file into the database.
    - `build time`: Display an indented tree view of completed tasks with their direct and rolled-up execution times.
    - `build rm <id:<task-id>|status:<status>>`: Safely delete a task (or group of tasks by status) recursively along with descendants, comments, and audit logs.
    - `build deploy-flow [flow-path]`: Deploy the specified Node-RED flow JSON file to the configured Node-RED server (defaults to `flows/sdlc-orchestrator-v1.json`).
    - `build sync-flows`: Bidirectionally synchronize visual workflows with Node-RED.
    - `build task <next|blocked|stuck|comments|update>`: Perform transaction-safe queries and modifications on tasks (used by Node-RED).
        - `next`: Fetch the next actionable leaf task JSON.
        - `blocked`: Fetch all blocked tasks JSON.
        - `stuck`: Fetch all stuck tasks JSON.
        - `comments --id <task-id>`: Retrieve comments for a task in JSON format.
        - `update --id <task-id> [options]`: Update task details (status, assignee, comment, etc.).
    - `build version`: Show the build tool version.

## 4. Hierarchy
- **Task**: Single, recursive entity representing any unit of work. Leaf nodes (tasks without children) are the ones that are actively worked on by the agents.
- **Approval Attempts**: Tracks how many times a task has failed testing or Boss validation. At 3 failures, it escalates to the Lead Engineer (stuck status). If the Lead Engineer has intervened 3 times already and the task fails again, it escalates to the Owner (failed status).
