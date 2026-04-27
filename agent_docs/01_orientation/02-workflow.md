# 02-Workflow

This document defines the deterministic rules for task management, agent interaction, and escalations within the `build` orchestrator.

## 1. The Core Lifecycle (Dev -> Tester -> Boss)

All actionable tasks (leaf nodes in the task tree) begin in the `todo` status and are processed in the following linear workflow, orchestrated entirely by the `Router`:

1. **Dev Assignment**: The Router assigns the task to the `Dev`. The Dev reads the context (`build context`), implements the feature/fix, and exits.
2. **Tester Assignment**: The Router hands the task to the `Tester`. The Tester reads the context, writes unit tests targeting the Dev's code, and exits. (If tests were added, the Tester updates `.build/test` to trigger them; otherwise, the default `exit 0` handles the pipeline).
3. **Automated Testing**: The Router runs the project's test suite via an adapter script (`./.build/test`).  
   - **Pass**: The task is advanced to the Boss.
   - **Fail**: The Router logs the test output as a comment, increments `approval_attempts`, and kicks the task back to the Dev.
4. **Boss Verification**: The `Boss` reviews the code and test output against the original task description.
   - The Boss runs `build comment <task-id> '<json>'` to provide its decision in a strict JSON format containing `"reasoning"` and `"approval": true/false`.
   - **Approve (`true`)**: The Router parses the JSON and marks the task as `done`.
   - **Reject (`false`)**: The Router increments `approval_attempts` and kicks the task back to the Dev.
   - **Format Error**: If the Boss fails to provide valid JSON, the Router catches this and kicks the task back to the Boss with a System Error explaining the mistake.

## 2. Agent Communication
Agents are not allowed to arbitrarily edit tasks. All notes, context, feedback, and test outputs are appended to the task chronologically using `build comment <task-id> "<comment>"`. Agents retrieve this history using `build context <task-id>`.

## 3. Escalation: The 3-Strike Rule
The `Router` keeps track of how many times a task has been kicked back to the Dev using the `approval_attempts` database column.

If a task fails automated testing OR Boss approval **3 times**:
1. The Router marks the task status as `failed`.
2. The Router assigns the task to the `Owner` (the human).
3. The Router **halts all operations** while a task is in a failed state.

**Human Intervention**: 
When a task fails, the Owner must review the context, provide explicit clarifying feedback via `build comment`, and then execute `build redo <task-id>`. This resets the task to `todo`, resets the `approval_attempts` to 0, and assigns it back to the Dev to run through the cycle again with the new human feedback.
