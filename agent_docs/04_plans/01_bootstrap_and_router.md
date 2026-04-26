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
  - `id` (INTEGER PRIMARY KEY), `parent_id` (INTEGER), `creator_id` (INTEGER - FK to agents), `assignee_id` (INTEGER - FK to agents), `title`, `description`, `status` ('todo', 'done'), `touch_count`, `escalation_level`.
- `audit_log`:
  - `id`, `task_id`, `actor_id` (FK to agents), `action`, `content`, `timestamp`.


## 3. Router Service & State Machine
The `Router` runs as a persistent background service, constantly reconciling the dependency tree.

### Status Model
- **`todo`**: Default state. Work must be done.
- **`done`**: Terminal state. All required sign-offs are complete.

### Reconciliation & Routing Logic
The `Router` constantly evaluates the dependency tree:
1.  **Dependency Check**: An entity is only 'actionable' if all children are in `status == 'done'`.
2.  **Assignment Engine**:
    - **Task**: Assigned to `Dev` -> `Tester` -> `Lead`.
    - **Lead Sign-off**: Once all tasks under an issue are `done`, the Lead evaluates. 
      - Pass: Issue -> `done`.
      - Fail: Issue -> `todo`, assigned back to `Dev` with comments.
    - **Escalation**:
      - Each level (Triad, Lead, Boss, Deputy) allows **3 escalations/rejections**.
      - If an entity is rejected for the 3rd time at a specific level, the Router automatically escalates it to the supervisor (Boss/Deputy/Owner).
3.  **Persistence**: The `Router` tracks `touch_count` and `escalation_level` per entity to trigger the escalation logic.

## 4. Work Delegation Hierarchy
- Owner (Goal) -> Deputy (Epic) -> Boss (Issue) -> Lead (Task).
- **Integrity Rule**: Instructions flow down. No agent can alter inherited instructions.
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
    - Light Blue: `escalation_level > 0`.
