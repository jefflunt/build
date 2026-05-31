# Design: Support the `build rm` Subcommand

## User Story
- **Headline**: Support deleting tasks by ID or status with recursive cascading and safety checks.
- **Problem Statement**: Over time, the orchestrator database can accumulate old or obsolete tasks. Users need a safe and precise command to clean up either single tasks (and their children) or bulk-delete tasks by status, without leaving orphan entries or broken hierarchies in the database.
- **Objective**: Implement a new `build rm` subcommand that accepts arguments of the form `status:<status>` or `id:<task-id>`. It must recursively locate the target tasks and all of their descendants, count them, present a confirmation prompt to the user, and upon confirmation, delete those tasks along with their associated comments and audit logs.
- **Expected Outcome**:
  - Running `build rm` without correct arguments or with invalid syntax prints usage instructions and exits with code `1`.
  - Running `build rm status:done` displays:
    ```text
    This will delete 5 tasks (including all descendants, comments, and audit logs).
    Are you sure you want to proceed? [y/N]: 
    ```
    If the user enters `y` or `Y`, it deletes the records and outputs:
    ```text
    Successfully deleted 5 tasks.
    ```
    Otherwise, it aborts the deletion and prints:
    ```text
    Deletion aborted.
    ```

## Implementation Backlog

### Pending
- `[CLI]` Route the `rm` subcommand in `cmd/build/main.go` and add its usage instruction to `build help`.
- `[LOGIC]` Implement task resolution and recursive descendant fetching logic for both `id:<task-id>` and `status:<status>` targets.
- `[DB]` Implement clean deletion queries that purge the resolved task IDs from `tasks`, `comments`, and `audit_logs` tables.
- `[CLI]` Implement confirmation prompting logic using `bufio.Scanner` to securely read `y`/`Y` from stdin.
- `[TEST-UNIT]` Write unit tests verifying that recursive cascading, count logic, and actual deletion purge the correct records while leaving other tasks intact.

### Current

### Completed

## Architecture Overview

**Database Operations:**
To ensure no orphaned records remain, when we delete a set of task IDs:
1. We recursively find all descendants of the target tasks. Let the complete set of unique task IDs to be deleted be `IDs`.
2. Delete from `tasks` where `id` is in `IDs`.
3. Delete from `comments` where `task_id` is in `IDs`.
4. Delete from `audit_logs` where `task_id` is in `IDs`.

**File Tree:**
All new logic will be isolated within `internal/db` or a new package `internal/rmcmd/` to keep `main.go` clean.
```text
cmd/build/
  main.go         # Route `rm` command
internal/
  rmcmd/          # NEW PACKAGE
    rmcmd.go      # Fetch, recursive search, prompt, and execute deletions
    rmcmd_test.go # Comprehensive unit tests
```

## Checklist & TDD Requirements
- `[TEST-UNIT]` Write tests for `ResolveTasksToDelete(db, target)` covering:
  - Deleting by specific ID with children (cascade verification).
  - Deleting by status (fetching matches + their descendants).
  - Deleting non-existent ID or empty status.
- `[TEST-UNIT]` Write tests for `DeleteTasks(db, ids)` verifying tasks, comments, and audit logs are deleted.
- Ensure the prompt functions can accept an `io.Reader` for testing user confirmation.

## Agent Instructions for Implementation
- Read-Analyze-Explain-Propose-HALT!
- Only edit one file at a time.
- Do not edit a file without a test.
- Prove tests pass before moving to the next file.
