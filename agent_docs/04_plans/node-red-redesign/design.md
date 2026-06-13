# Design: Node-RED Visual Workflow Redesign (Hub-and-Spoke & HTTP Trigger)

## User Story
* **Headline**: Transform Node-RED Workflow to On-Demand Event-Triggered Hub-and-Spoke Architecture
* **Problem Statement**: The current Node-RED flow utilizes a heavy 5-second polling interval and embeds raw `sqlite3` child-process shell commands. This results in heavy idle CPU usage, shell-formatting vulnerabilities, and tight coupling with SQLite. Furthermore, the direct routing linkages make it hard to visually understand task execution state.
* **Objective**: Remove polling entirely. Expose an HTTP `/trigger-build` listener in Node-RED and trigger it via the `build` CLI on-demand. Redesign the flow to use a Hub-and-Spoke canvas, and refactor all modular subflows to query and update tasks using the `build task` CLI endpoints.
* **Expected Outcome**: Instantaneous, event-driven build execution with zero idle CPU overhead, clean visual tracing in Node-RED, and a 100% database-blind Node-RED workflow.

---

## Architecture Overview

### 1. The Triggering Chain (Event-Driven / Push)
When a user enqueues, redos, or starts the workflow via the CLI, the CLI sends a quick HTTP POST request to Node-RED's endpoint.

```
+---------------+      HTTP POST /trigger-build      +------------------+
|   build CLI   | ==================================>|     Node-RED     |
| (ingest/redo) |                                    | (HTTP Listener)  |
+---------------+                                    +------------------+
```

We will modify:
- `build start`: Instead of running the Go-based background router, it will send a trigger request to Node-RED.
- `build ingest` & `build redo`: Will automatically trigger Node-RED after task changes.

### 2. The Hub-and-Spoke Flow Layout
The main canvas `flows/sdlc-orchestrator-v1.json` will implement a Hub dispatch logic:

1. **Trigger Endpoint (`/trigger-build`)**: Receives POST, immediately returns `200 OK` (to unblock the CLI), and forwards the payload.
2. **Locking Check**: Check the `is_processing` global/flow lock. If locked, abort. If free, set lock.
3. **Task Pull**: Executes `build task next`.
4. **Router Dispatcher (Hub)**:
   - If JSON is empty `{}`: release lock and stop.
   - If JSON is valid: inspect assignee (`agent_id`) and pass to the corresponding Spoke (Subflow).
5. **Spoke Execution**: The task passes into the specific modular subflow (`Dev`, `Tester`, `Boss`, etc.) which performs its work, updates status via `build task update`, and returns back to the Hub.
6. **Next Loop**: The Hub queries `build task next` again, repeating until no more tasks exist.

### 3. Subflows Migration (Refactoring from `sqlite3` to `build task update`)
We will rewrite all modular subflows under `subflows/` to replace `sqlite3` commands with clean `exec` nodes calling `build task update`.

---

## Checklist & TDD Requirements

1. **Extending `TaskJSON`**:
   - Update `TaskJSON` struct in `cmd/build/main.go` to include `ApprovalAttempts` and `LeadInterventions`. This allows subflows (like Boss or Tester) to read attempt counts directly from the task payload.
   - Run `go test` to verify no regressions.
2. **HTTP Trigger Connection**:
   - Write Go test in `cmd/build/main_test.go` confirming that `build start` successfully calls a mock Node-RED HTTP endpoint.
3. **Workflow Integrity**:
   - Refactor flow/subflows files on disk and use `build sync-flows` to deploy them to Node-RED.
   - Validate that Node-RED accepts the newly merged workflows.

---

## Implementation Backlog

## Pending

## Current

## Completed
- Task 1: Extend the Go 'TaskJSON' struct in 'cmd/build/main.go' to include fields for 'approval_attempts' and 'lead_interventions', and update 'runTaskQuery' to query and scan these values.
- Task 2: Implement an HTTP POST notification trigger within 'cmd/build/main.go' and integrate it into 'start', 'ingest', and 'redo' command-line routing paths.
- Task 3: Refactor the main canvas orchestrator workflow in 'flows/sdlc-orchestrator-v1.json' to implement a Hub-and-Spoke layout containing a '/trigger-build' HTTP listener.
- Task 4: Rewrite the seven modular subflows under 'subflows/' to execute status and assignee updates through 'build task update' CLI command nodes instead of raw 'sqlite3' CLI commands.
