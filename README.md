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
4.  **Ingestion**:
    ```bash
    # Ingest the breakdown output into the system
    build ingest <session-id>
    ```

## 4. Monitoring Progress (`build route watch`)
To observe the work as it progresses, use the TUI dashboard.

```bash
# Start the Router in the background first
build start

# Open the dashboard
build route watch
```

### Dashboard Color Coding
- **Green**: `done`
- **Grey**: `todo` (Parent with incomplete children)
- **Yellow**: `todo` (Ready to be worked on/signed off)
- **Purple**: Active (`assigned` to an agent)
- **Light Blue**: Escalated (`escalation_level > 0`)

## 5. Escalation and Sign-off
You are shielded from the day-to-day work. The system handles 3-strike deadlocks automatically via the `Router`. You only interact with `Boss-X` agents directly when:
1. You are delegating new work (`build design`).
2. A `Boss-X` requires clarification on high-level strategy that they cannot resolve themselves.
3. The project goal is complete and ready for your final sign-off.
