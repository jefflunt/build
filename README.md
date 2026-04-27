# Getting Started with `build`

Welcome to `build`. This guide provides the essentials for the Owner to initialize projects, design work, and maintain oversight of the agentic workflow.

## 1. Build, Test, and Install
We adhere to a standard script-based workflow in the `/script` directory. To prepare the project:

```bash
# Build, Test, and Install
./script/build-test-install
```

## 2. Initialization (`build init`)
Before starting a project, initialize the local environment. This creates the `.build/` directory and prepares the initial SQLite state, including the Owner agent (id=1).

```bash
build init
```

## 3. Workflow: Designing Work
The entry point for all new work is a design session.

1.  **Start a Design Session**:
    ```bash
    build design
    ```
2.  **Interaction**: You will be prompted for your goal. An `opencode` session (using the `designer` agent) will spawn, utilizing the "Grill Me" protocol to vet your ideas.
3.  **Decomposition**: Once the session exits and a `design.md` file is generated, the breakdown process automatically structures the work into a recursive task tree.
4.  **Ingestion**: You can manually ingest external breakdowns using:
    ```bash
    # Ingest the breakdown output into the system by pointing to its root directory
    build ingest ~/.breakdown/output/tic-tac-toe/
    ```

## 4. Monitoring Progress
To observe the work as it progresses, start the router service in the background:

```bash
# Start the Router in the background
build start

# Show router status
build status
```

The Router handles moving tasks through the `Dev` -> `Tester` -> `Boss` workflow automatically. It uses color-coded output:
- **Medium Grey**: `done`
- **Light Blue**: `active` (`assigned` to `Dev`)
- **Light Yellow**: `active` (`assigned` to `Tester`)
- **Light Green**: `active` (`assigned` to `Boss`)
- **Red**: `failed` (Requires human intervention)

## 5. Escalation and Sign-off
You are shielded from the day-to-day work. The system handles retries and deadlocks automatically via the `Router`. You only interact with the system when:
1. You are delegating new work (`build design` & `build ingest`).
2. A task reaches `approval_attempts >= 3` and transitions to the `failed` state. When this happens, the Router halts and assigns the task to the `Owner` (you) so you can review the comments via `build context <task-id>`, offer feedback via `build comment <task-id> "<comment>"`, and run `build redo <task-id>` to restart the workflow.

## 6. Other Useful Commands
- `build context <task-id>`: View the task description and history of comments.
- `build comment <task-id> "<comment>"`: Leave a note or feedback on a task.
- `build approve <task-id>`: Manually mark a task as `done` (only used if the Boss fails to run it).
- `build redo <task-id>`: Reset a `failed` task.
