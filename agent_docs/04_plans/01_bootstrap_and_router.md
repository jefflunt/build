# Plan 01: Initial Bootstrap and Router Engine

## Goal
Establish the foundational project structure, set up the SQLite persistence layer, and implement the persistent `Router` service.

## 1. Environment Setup
- Language: **Go 1.22+**.
  - *Justification*: Compiles to efficient, cross-platform native binaries. Excellent concurrency model for the `Router` engine and robust standard library for systems programming.
- **CLI Design Guidelines** (per `jefflunt/cli-guidelines`):
  - Sub-commands preferred over flags (e.g., `build help`, `build run`).
  - Help is a sub-command (`build help`).
  - Configuration: Default `~/.build/config.yml`, overrideable via `-c`.
  - Streams: `stdout` for data/results, `stderr` for logs/warnings.
  - Standard folders: `cmd/`, `internal/`, `pkg/`, `script/`.

- Project structure:
  ```
  /build
    /cmd
      /build       # CLI entry point
    /internal
      /router      # The persistent Router engine
      /db          # SQLite schema and access logic
      /agents      # Base agent framework
    /script        # build, test, install scripts
    /data          # Persistent SQLite storage
    /agent_docs    # Documentation
  ```

## 2. SQLite Schema Design
- `goals`, `epics`, `issues`, `tasks` tables.
- `state_machine` table (to track current state and assigned agent for each unit).
- `audit_log` table (for event-sourced state changes/conversations).

## 3. Router Service Skeleton
- Implement a persistent Python background service.
- Use `pidfile` for management.
- Basic loop:
  1. Query `tasks` needing attention (status = 'pending').
  2. Map tasks to the appropriate agent role (Dev/QualityEngineer/Lead).
  3. Update `state_machine` and trigger agent.

## 4. Verification
- Validate schema creation.
- Start/stop the `Router` service and verify via PID file.
- Add a basic test task and verify the `Router` detects it.
- **Triad Team**: Lead, Dev, Tester.
