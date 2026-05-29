# Getting Started with `build` 🛠️

Welcome to `build`! This guide is a complete, step-by-step user manual designed to help you sit down, design a new project or feature, and let your automated agent workspace build it for you.

`build` is a lightweight, autonomous multi-agent task orchestrator. It manages the lifecycle of your project by coordinating specialized AI agents to write code, write tests, run automated verifications, and evaluate completion against your initial specifications.

---

## The Crew: Who's Building Your Code?

When you run `build`, you sit at the center of an elite digital product team:
- **Owner (You - ID 1)**: The ultimate decision-maker and visionary. You define goals, write plans, provide course-correcting feedback, and sign off when needed.
- **Developer (Dev - ID 2)**: The execution engine. Dev reads task instructions, writes production code, and moves the code to testing.
- **Tester (ID 3)**: The quality gatekeeper. Tester inspects Dev's code and writes fast, isolated unit tests to ensure it behaves correctly.
- **Lead Engineer (ID 5)**: The troubleshooter. If Dev gets stuck or tests fail multiple times, the Lead Engineer steps in automatically to analyze logs and leave structural comments guiding Dev.
- **Boss (ID 4)**: The project manager. Boss reads the original task specifications, reviews the code and test suite output, and determines whether the task is fully complete.
- **Router (The Orchestrator)**: The background engine that persistent-polls your project database, launches agent sessions, manages source control commits, runs your test adapter, and routes tasks through the lifecycle.

---

## 🚀 Step-by-Step Guide: Sit Down and Make Something

Follow these steps to build your first feature or application with `build`.

### Step 1: Initialize Your Project Workspace
Create a directory for your new project and run `build init` to initialize the workspace.

```bash
# Create and move into your project directory
mkdir my-app && cd my-app

# Initialize the build environment
build init
```

**What just happened?**
`build init` created a hidden `.build/` directory in your workspace containing:
1. `build.db`: A SQLite database tracking tasks, agents, audit logs, and comment history.
2. `test`: A default shell script acting as an adapter between the orchestrator and your project's test suite (defaults to `exit 0`).

---

### Step 2: Write Your Plan (The Blueprint)
`build` consumes work through structured Markdown plans. Create a plan file (e.g., `my-feature.md`) with a **Goal** and a hierarchical breakdown of **Tasks**.

Create `my-feature.md`:
```markdown
# Goal
Build a fast, lightweight JSON calculator CLI tool.

# Tasks
## Composite: Project Scaffolding
- Atomic: Initialize Go module and setup main entry point
- Atomic: Setup standard input/output handling

## Composite: Core Calculator Logic
- Atomic: Implement additions and subtractions
- Atomic: Implement multiplications and divisions
- Atomic: Handle divide-by-zero errors gracefully and return JSON errors

## Composite: CLI Parsing
- Atomic: Parse mathematical operations from JSON-formatted command-line flags
```

> **Naming Rule**: Prepend task titles with `Composite:` for groupings/headers and `Atomic:` for actual leaf-node tasks that agents should execute.

---

### Step 3: Enqueue Your Plan
To load your plan into the SQLite database, use the `enqueue` subcommand:

```bash
build enqueue my-feature.md
```

**What just happened?**
`build` automatically invoked the `breakdown` utility behind the scenes. This parses your markdown file, breaks it down into individual task files, and ingests them into `build.db` as a hierarchical task tree. Your tasks are now staged in the `todo` state and ready for development.

---

### Step 4: Configure Your Test Suite (The Adapter)
Before starting, you must tell `build` how to run your tests. Specialized agents write unit tests, and the orchestrator runs them using the adapter script at `.build/test`.

Open `.build/test` in your text editor and point it to your real test command:

```bash
#!/usr/bin/env bash
# Update this file to execute your project's actual test runner.
# Examples:
#   go test ./...
#   npm test
#   pytest

go test ./...
```

⚠️ **The 3 Golden Rules of Testing in `build`:**
1. **The 30-Second Timeout**: The orchestrator enforces a strict **30-second execution limit** on your test adapter.
2. **Strict Isolation**: Integration tests are not supported here. Your tests must be fast unit tests that use mocks/stubs instead of hitting real databases, networks, or file systems.
3. **Exit Codes**: The test script must return `0` on success, and a non-zero code on failure.

---

### Step 5: Start the Orchestrator Loop
Now, run the persistent background router. This initiates the multi-agent development pipeline!

```bash
build start
```

This starts a terminal process showing a live, color-coded task tree updating in real-time as tasks are worked on:
- **Medium Grey**: Finished tasks (`done`)
- **Light Blue**: Handed to the **Dev** (ID 2) to implement
- **Light Yellow**: Handed to the **Tester** (ID 3) to write tests
- **Light Green**: Handed to the **Boss** (ID 4) to verify and sign off
- **Orange**: Handed to the **Lead Engineer** (ID 5) to resolve blockers
- **Red**: Failed tasks requiring **Your** intervention (ID 1)

The Router automatically runs tests after the Tester completes their work. If they pass, the task moves to the Boss. If they fail, the test log is saved as a task comment, and the task is kicked back to the Dev.

---

### Step 6: Handle Escalations & The 3-Strike Rule
You are shielded from day-to-day coding. The Router automatically handles retries and pipelines. However, if a task fails automated tests or Boss review multiple times:

1. **Strike 1 & 2**: If a task fails twice, the Router automatically escalates it to the **Lead Engineer** (ID 5) who inspects the failures, diagnoses the issue, and leaves high-level comments to guide the Developer.
2. **Strike 3 (Hard Failure)**: If the task fails a 3rd time, the Router flags the status as `failed`, assigns it to the **Owner (You)**, and **halts all operations** to prevent infinite agent loops.

#### How to Unblock a Failed Task:
1. **Inspect the logs**:
   ```bash
   build why-failed <task-id>
   ```
   This command gives you the full audit timeline and complete comment history (including test outputs, Boss reasoning, and Lead Engineer suggestions) for that task.

2. **Give feedback**:
   Leave a clarifying comment or instruction for the Developer:
   ```bash
   build comment <task-id> "Make sure you read from standard input in a loop, rather than doing a single-read buffer."
   ```

3. **Reset and Resume**:
   Restart the automated cycle:
   ```bash
   build redo <task-id>
   ```
   This resets the task status to `todo`, resets the strike counter to `0`, and assigns it back to the Developer to try again with your new guidance.

---

### Step 7: Aggregating and Tracking Development Time
Want to see where the development hours were spent? `build` logs every second of agent activity into the database. You can view a rolled-up time tree at any time:

```bash
build time
```

This returns an indented tree view showing:
`[TotalTimeSpent] / [DirectTimeSpent] Task Title`
```text
[1h23m05s] / [0h00m00s] Project Scaffolding
  [0h45m10s] / [0h00m00s] Initialize Go module and setup main entry point
  [0h37m55s] / [0h15m20s] Setup standard input/output handling
```

---

## 📋 CLI Command Quick Reference

| Command | Usage | Description |
| :--- | :--- | :--- |
| **`init`** | `build init` | Prepares the `.build/` repo state, seeds initial SQLite database, and templates. |
| **`enqueue`** | `build enqueue <plan-file>` | Decomposes and ingests a high-level Markdown plan into the task queue. |
| **`start`** | `build start` | Runs the persistent background multi-agent orchestrator service. |
| **`status`** | `build status` | Shows whether the background Router service is currently running. |
| **`context`** | `build context <task-id>` | View task title, description, and chronological comment history. |
| **`why-failed`** | `build why-failed <task-id>` | View the deep audit log, duration times, and comment history for a stalled task. |
| **`comment`** | `build comment <task-id> "<comment>"` | Post feedback/instructions to a task (automatically visible to agents on next run). |
| **`redo`** | `build redo <task-id>` | Resets a `failed` task back to `todo` status with 0 attempts. |
| **`time`** | `build time` | Displays a hierarchical, rolled-up time tree of all completed tasks. |
| **`teardown`** | `build teardown` | Safely removes the `.build/` database and configuration folder. |
| **`version`** | `build version` | Displays the current installed version of `build`. |

Enjoy building with `build`! 🚀
