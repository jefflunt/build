# 02-Workflow

This document defines the deterministic rules for task management, agent interaction, and escalations within the `build` orchestrator.

## 1. The Core Lifecycle (Dev -> Tester -> Boss)

All actionable tasks (leaf nodes in the task tree) begin in the `todo` status and are processed in the following linear workflow, orchestrated entirely by the Node-RED Router using the transaction-safe `build task` CLI command group:

1. **Dev Assignment**: The Router assigns the task to the `Dev`. The Router reads the task description and comment history from the database and injects it directly into the Dev's instructions. The Dev implements the feature/fix and exits.
2. **Tester Assignment**: The Router hands the task to the `Tester`, again injecting full context. The Tester writes unit tests targeting the Dev's code and exits. (If tests were added, the Tester updates `.build/test` to trigger them; otherwise, the default `exit 0` handles the pipeline).
3. **Automated Testing**: The Router runs the project's test suite via an adapter script (`./.build/test`).  
   - **Pass**: The task is advanced to the Boss.
   - **Fail**: The Router logs the test output as a comment, increments `approval_attempts`, and kicks the task back to the Dev.
4. **Boss Verification**: The `Boss` reviews the code and test output against the original task description (injected by the Router).
   - The Boss executes `build review <task-id> <approve|reject> "<reasoning>"` to provide its decision. The reasoning must be a single line of text enclosed in double quotes, using single quotes internally.
   - **Approve**: The Router marks the task as `done`.
   - **Reject**: The Router increments `approval_attempts` and kicks the task back to the Dev.
   - **Format Error**: If the Boss fails to use the command correctly, the Router catches this and kicks the task back to the Boss with a System Error explaining the mistake.
5. **Git Sweep and Commit**: After each agent finishes a session (such as Dev, Tester, Boss, or Lead), the Router automatically runs the `Git Cleanup Artist / Sweep` agent (id=6) to ensure the `.gitignore` is up-to-date and any unwanted build metadata, caches, or files are ignored. Following this, the Router stages all files (`git add .`) and commits changes with a standardized message: `build: updates for task <task-id> by <role>`.

## 2. Agent Communication
Agents are not allowed to arbitrarily edit tasks. All notes, context, feedback, and test outputs are appended to the task chronologically using `build comment <task-id> "<comment>"`. Agents retrieve this history automatically via the Router's context injection, but human Owners can view it using `build context <task-id>`.

## 3. Escalation: The 3-Strike Rule with Lead Engineer Escalation
The `Router` keeps track of how many times a task has been kicked back to the Dev using the `approval_attempts` database column and how many times the Lead Engineer has intervened using the `lead_interventions` database column.

If a task fails automated testing OR Boss approval **3 times**:

### 1. Lead Engineer Escalation
If `lead_interventions` is less than 3:
- The Router marks the task status as `stuck` and assigns the task to the `Lead Engineer` (id=5).
- **Lead Engineer Intervention**: The Lead Engineer reviews the full context and comment history to diagnose why the task is deadlocked. The Lead then leaves code-level or architectural instructions as a comment using `build comment <task-id> "..."` and exits.
- Upon completion of the Lead's review, the Router transitions the task status back to `todo`, resets `approval_attempts` to 0, increments `lead_interventions` by 1, and assigns it back to the `Dev` (id=2).

### 2. Hard Failure & Owner Escalation
If the task has already had 3 Lead Engineer interventions and fails automated testing or Boss approval 3 more times:
- The Router marks the task status as `failed` and assigns the task to the `Owner` (id=1, the human).
- The Router **halts all operations** while any task is in a failed state.

**Human Intervention**: 
When a task fails, the Owner must review the context (`build context <task-id>`), provide explicit clarifying feedback via `build comment`, and then execute `build redo <task-id>`. This resets the task to `todo`, resets the `approval_attempts` to 0, and assigns it back to the Dev to run through the cycle again with the new human feedback.
