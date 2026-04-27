# Plan 01: Initial Bootstrap and Router Engine

## Goal
Establish the foundational project structure, set up the SQLite persistence layer, and implement the persistent `Router` service.

## 1. Environment Setup
- Language: **Go 1.22+**.
- **CLI Design Guidelines** (per `jefflunt/cli-guidelines`):
  - Sub-commands preferred: `build help`, `build start`, `build status`.
  - Config: `~/.build/config.yml` (override via `-c`).
  - Streams: `stdout` for data, `stderr` for logs.
- Project Structure:
  ```
  /build
    /cmd/build        # Entry point
    /internal/router  # Persistent Router engine
    /internal/db      # SQLite schema and access
    /internal/agents  # Base agent framework
    /script           # build, test, install
    /data             # SQLite DB
  ```

## 2. SQLite Schema Design
- `agents` table: 
  - `id` (INTEGER PRIMARY KEY), `role` (TEXT), `name` (TEXT).
- `tasks` table (recursive): 
  - `id` (INTEGER PRIMARY KEY), `parent_id` (INTEGER), `agent_id` (INTEGER - FK to agents), `title`, `description`, `status` ('todo', 'done', 'failed'), `touch_count`, `approval_attempts`.
- `audit_logs`:
  - `id`, `task_id`, `actor_id` (FK to agents), `action`, `llm_provider`, `llm_model`, `llm_instructions_sha256`, `build_version`, `duration_seconds`, `timestamp`.

## 3. Router Service & State Machine
The `Router` runs as a persistent background service, constantly reconciling the dependency tree.

### Status Model
- **`todo`**: Default state. Work must be done.
- **`done`**: Terminal state. All required sign-offs are complete.

### Reconciliation & Routing Logic
The `Router` constantly evaluates the dependency tree:
1.  **Dependency Check**: An entity is only 'actionable' if it has no children, or all children are in `status == 'done'`. Stop completely if any task is `status == 'failed'`.
2.  **Assignment Engine**:
    - **Task**: Assigned linearly `Dev` -> `Tester` -> `Boss`.
    - **Automated Tests**: Tested between Dev and Boss. Failures kick back to Dev.
    - **Sign-off**: Boss evaluates against intent. Approve sets to `done`. Rejection kicks back to Dev via `build comment`.
3. **Escalation**:
    - If a task is kicked back (fails tests or Boss rejects) 3 times (`approval_attempts >= 3`), the Router transitions it to `failed`.
    - At this point, the human Owner must intervene, leave comments, and run `build try_again <task-id>` to restart the cycle.

## 4. Work Delegation Hierarchy
- Human Owner -> Boss -> Tester/Dev.
- **Integrity Rule**: Instructions flow down. No agent can alter inherited instructions. They can only append to the task history via `build comment`.
- **Bi-directional Flow**:
  - Breakdown: Instructions flow down.
  - Completion/Escalation: Feedback (pass/fail/clarification) flows up.

## 6. CLI Observability (TUI)
- Implementation: `build route watch`.
- **View Logic**:
  - Header: Project Name, Summary (Todo/Done/Total).
  - Tree Rendering:
    - Green: `done`.
    - Grey: `todo` with children.
    - Yellow: `todo` ready (leaf node or dependencies done).
    - Purple: `assigned` (active).
    - Red: `failed` (Requires human intervention).
