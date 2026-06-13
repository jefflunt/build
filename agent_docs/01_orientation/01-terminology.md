# 01-Terminology

This document defines the core entities and architectural concepts for the `build` orchestrator.

## 1. The Human & Orchestrator
- **Owner**: Human (id=1), ultimate decision maker/instruction giver.
- **Router**: Deterministic background service; the *only* entity authorized to spawn agent sessions, assign agents, trigger escalations, or run tests.

## 2. The Core Agents
- **Dev** (id=2): Implements the task according to its instructions.
- **Tester** (id=3): Verifies the Dev's work by writing test cases.
- **Boss** (id=4): Evaluates the completed implementation and test suite output to verify it fulfills the entire original intent of the task.

## 3. Operations
- **Router Process**: Hosted inside Node-RED as an on-demand, event-driven orchestrator, triggered instantly on task changes (e.g. starting, ingesting, or redoing tasks) via HTTP POST.
- **CLI Commands**:
    - `build init`: Initialize local `.build/` repo state.
    - `build design`: Spawn a `build-designer` agent for a new project/goal.
    - `build ingest <path>`: Import external `breakdown` output directory into the system database.
    - `build start`: Writes a local PID and fires an HTTP POST notification to trigger the Node-RED orchestrator on-demand, blocking until stopped.
    - `build status`: Check if the `Router` service is running.
    - `build context <task-id>`: Retrieve task details and history.
    - `build comment <task-id> "<comment>"`: Append a comment to a task.
    - `build approve <task-id>`: Manually mark a task as `done`.
    - `build redo <task-id>`: Reset a `failed` task for the Dev.
    - `build task <next|blocked|stuck|comments|update>`: Perform transaction-safe queries and modifications on tasks (used by Node-RED).
    - `build time`: Display an indented tree view of completed tasks with their direct and rolled-up execution times.
    - `build rm <id:<task-id>|status:<status>>`: Safely delete a task (or group of tasks by status) recursively along with descendants, comments, and audit logs.

## 4. Hierarchy
- **Task**: Single, recursive entity representing any unit of work. Leaf nodes (tasks without children) are the ones that are actively worked on by the agents.
- **Approval Attempts**: Tracks how many times a task has failed testing or Boss validation. At 3 failures, it escalates to the Owner.
