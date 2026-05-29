# Getting Started with `build`

`build` is a lightweight, autonomous multi-agent task orchestrator. It coordinates a pipeline of specialized agents to implement features, write unit tests, run automated verifications, and evaluate completion against a structured markdown plan.

---

## Core Concepts and Agent Roles

The orchestrator manages the lifecycle of each task through a sequence of assignments to specific agents:

- **Owner (ID 1)**: The human operator who initializes the project, defines goals/plans, reviews failures, and provides feedback when needed.
- **Developer (Dev - ID 2)**: Receives task instructions, writes or modifies implementation code, and hands the task off for testing.
- **Tester (ID 3)**: Inspects the developer's changes and writes automated unit tests targeting the implementation.
- **Lead Engineer (ID 5)**: If a task fails verification multiple times, the Lead Engineer automatically reviews logs and leaves corrective structural guidance for the Developer.
- **Boss (ID 4)**: Evaluates the implementation and test results against the original task specifications to approve or reject the work.
- **Router**: The persistent background process that monitors the task database, triggers agent execution via `opencode` CLI sessions, manages git commits, and executes the test adapter.

---

## Practical Workflow: Building a Feature

This guide walks through the standard lifecycle of initializing a workspace, writing a plan, configuring tests, and running the orchestrator.

### 1. Initialize the Project
Create a project directory and run `build init` to establish the environment:

```bash
mkdir my-app && cd my-app
build init
```

The command creates a hidden `.build/` directory containing:
- `build.db`: A SQLite database tracking tasks, agent assignments, comment history, and audit logs.
- `test`: A default shell script adapter for the project's test suite.

---

### 2. Write the Plan
Plans must be written in Markdown format. A plan is defined by a single high-level Goal and a list of hierarchical Tasks.

Create `my-feature.md`:
```markdown
# Goal
Build a JSON-based calculator CLI tool in Go.

# Tasks
## Composite: Project Scaffolding
- Atomic: Initialize Go module and setup main entry point
- Atomic: Setup standard input/output handling

## Composite: Core Calculator Logic
- Atomic: Implement addition and subtraction functions
- Atomic: Implement multiplication and division functions
- Atomic: Handle divide-by-zero errors and return formatted JSON error output

## Composite: CLI Parsing
- Atomic: Parse mathematical operations from JSON flags
```

*Note: Grouping headers should be prefixed with `Composite:`, and leaf-node tasks should be prefixed with `Atomic:`.*

---

### 3. Enqueue the Plan
Load the plan file into the local task queue:

```bash
build enqueue my-feature.md
```

This command parses the plan, generates a hierarchical task tree using the `breakdown` utility, and stores the resulting tasks in `build.db` with a status of `todo`.

---

### 4. Configure the Test Adapter
Agents write and verify tests using the local adapter script at `.build/test`. Open `.build/test` and update it to call your project's test runner:

```bash
#!/usr/bin/env bash
# .build/test - Adapt this script to run your test suite.
# Ensure it exits 0 on success, and non-zero on failure.

go test ./...
```

**Key Execution Constraints:**
- **30-Second Timeout**: The Router enforces a strict 30-second timeout on the test adapter.
- **Isolation**: Tests must run fast and remain isolated. Avoid real network or database calls; use mocks/stubs.
- **Exit Status**: The test script must return `0` on success, and a non-zero exit code on failure.

---

### 5. Start the Orchestrator
To process enqueued tasks, run the background loop:

```bash
build start
```

This command starts a live console display showing the task tree status represented by active agents:
- **Grey**: Task is complete (`done`).
- **Blue**: Assigned to the **Developer** (ID 2).
- **Yellow**: Assigned to the **Tester** (ID 3).
- **Green**: Assigned to the **Boss** (ID 4) for final evaluation.
- **Orange**: Assigned to the **Lead Engineer** (ID 5) for structural diagnosis.
- **Red**: Task has failed and requires **Owner** (ID 1) intervention.

The Router automatically runs tests after the Tester completes their work. If the tests pass, the task moves to the Boss. If they fail, the output is saved as a task comment, and the task is assigned back to the Developer.

---

### 6. Handling Task Failures & Escalations
The system operates on an automated loop but stops to escalate hard blockers to prevent infinite retries.

#### The Escalation Pipeline:
1. **First & Second Failures**: If a task fails unit testing or Boss verification, the Router assigns it back to the Developer.
2. **Third Failure**: The Router escalates the task to the **Lead Engineer** (ID 5). The Lead Engineer analyzes the error logs and leaves clarifying guidance.
3. **Subsequent Hard Blockers**: If the task remains blocked, it enters the `failed` state and is assigned to the **Owner (You)**. The Router halts execution until you intervene.

#### Resolving a Failed Task:
1. **Audit the failure**:
   ```bash
   build why-failed <task-id>
   ```
   This outputs the task's complete history, audit log, duration, and comment thread.

2. **Provide clarification**:
   Add feedback or hints directly to the task context:
   ```bash
   build comment <task-id> "Read standard input in a loop instead of a single buffered read."
   ```

3. **Reset the workflow**:
   Instruct the system to resume:
   ```bash
   build redo <task-id>
   ```
   This resets the task status to `todo`, clears the strike counter, and assigns it back to the Developer.

---

### 7. Tracking Development Time
The system logs the exact duration of each agent's execution. To view a rolled-up view of the time spent across the task hierarchy, run:

```bash
build time
```

Output format: `[Total Time] / [Direct Time] Task Title`
```text
[1h23m05s] / [0h00m00s] Project Scaffolding
  [0h45m10s] / [0h00m00s] Initialize Go module and setup main entry point
  [0h37m55s] / [0h15m20s] Setup standard input/output handling
```

---

## CLI Command Reference

| Subcommand | Syntax | Description |
| :--- | :--- | :--- |
| **`init`** | `build init` | Initializes the `.build/` workspace, SQLite database, and configuration templates. |
| **`enqueue`** | `build enqueue <plan-file>` | Decomposes and ingests a Markdown plan into the task queue. |
| **`start`** | `build start` | Runs the persistent background multi-agent orchestrator loop. |
| **`status`** | `build status` | Checks if the background Router service is currently running. |
| **`context`** | `build context <task-id>` | Displays a task's description and chronological comment history. |
| **`why-failed`** | `build why-failed <task-id>` | Shows a detailed audit timeline, agent durations, and comment history for a task. |
| **`comment`** | `build comment <task-id> "<comment>"` | Appends a comment to a task (injected as context for agents on future runs). |
| **`redo`** | `build redo <task-id>` | Resets a `failed` task back to `todo` status and resets attempts to 0. |
| **`time`** | `build time` | Displays a hierarchical tree of time spent on completed tasks. |
| **`teardown`** | `build teardown` | Removes the `.build/` database and environment files. |
| **`version`** | `build version` | Prints the installed build version of the executable. |
