# Implement `build time` Command

## Problem Statement
The user needs a way to view the total accumulated execution time of tasks within the `build` orchestrator database. Currently, the system logs `duration_seconds` in the `audit_logs` table for individual agent actions, but there is no CLI tool to aggregate and display this data hierarchically to understand the "cost of completion."

## Objective
Implement a new `build time` subcommand that outputs an indented, human-readable tree view of all tasks marked as `done`. For composite tasks (tasks with children), the output must display both the time spent directly on the task and the rolled-up sum of time spent on all its descendants.

## Expected Outcome
The user can run `build time` and see an output similar to:
```
[1h23m05s] / [0h00m00s] Task Title
  [0h45m10s] / [0h00m00s] Child Task A
  [0h37m55s] / [0h15m20s] Child Task B
    [0h22m35s] / [0h00m00s] Grandchild Task 1
```
(Format logic: `[TotalTime] / [DirectTime] Title`)

## Implementation Backlog

### Pending
- `[CLI]` Update `cmd/build/main.go` help text to include the `time` subcommand.
- `[CLI]` Update `cmd/build/main.go` `runCLI` switch statement to route `"time"` to a new `printTime()` function.
- `[DB]` Implement database query logic to fetch all tasks where `status = 'done'`.
- `[DB]` Implement database query logic to sum `duration_seconds` from `audit_logs` grouped by `task_id`.
- `[LOGIC]` Create a recursive function to build the tree structure and calculate the rolled-up times.
- `[LOGIC]` Create a formatting function to convert integer seconds into `HhMMmSSs` format (e.g., `0h05m09s`, `100h00m00s`).
- `[CLI]` Implement the recursive rendering function to print the indented tree view to standard out.
- `[TEST-UNIT]` Write tests for the time formatting function.
- `[TEST-UNIT]` Write tests for the tree calculation logic.

### Current
- 

### Completed
- 

## Architecture Overview

**File Tree:**
```
cmd/build/
  main.go         # CLI entrypoint and routing
internal/
  db/
    db.go         # Database connection logic
  timecmd/        # NEW PACKAGE
    timecmd.go    # Logic for fetching, tree-building, and rendering
    timecmd_test.go
```

**Data Flow:**
1. User executes `build time`.
2. `cmd/build/main.go` catches the subcommand and calls `timecmd.Run()`.
3. `timecmd.Run()` connects to the database via `internal/db`.
4. It queries the `tasks` table for all `done` tasks and the `audit_logs` table for the sum of durations.
5. It constructs a memory representation of the task tree.
6. A recursive function walks the tree from the roots, calculating the rolled-up time (`RollupTime = DirectTime + Sum(Children's RollupTime)`).
7. The tree is printed to standard out with proper indentation and time formatting (`%dh%02dm%02ds`).

## Checklist & TDD Requirements
- `[TEST-UNIT]` Create `internal/timecmd/timecmd_test.go` and write a test for the time formatter (e.g., asserting `3665` seconds becomes `1h01m05s`).
- `[TEST-UNIT]` Write a test for building the tree and rolling up times, using mock data.
- Ensure all logic is isolated in a new package `internal/timecmd` to prevent bloating `main.go`.
- Make sure to ONLY query tasks that have `status = 'done'`.
- If a composite task is `done` but has children that are not `done`, ONLY include the `done` children in its rollup time. 

## Agent Instructions for Implementation
- Read-Analyze-Explain-Propose-HALT!
- Only edit one file at a time.
- Do not edit a file without a test.
- Prove tests pass before moving to the next file.