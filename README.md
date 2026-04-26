# Getting Started with `build`

Welcome to `build`. This guide provides the essentials for the Owner to initialize projects, delegate goals, and maintain oversight of the agentic workflow.

## 1. Build, Test, and Install
We adhere to a standard script-based workflow in the `/script` directory. To prepare the project:

```bash
# Build the binary
./script/build

# Run the test suite
./script/test

# Install to your local path
./script/install
```

## 2. Initialization (`build init`)
Before starting a project, initialize the local environment. This creates the `.build/` directory and prepares the initial SQLite state, including the Owner agent.

```bash
build init
```

## 3. Workflow: Delegating Goals
The entry point for all new work is a conversation with a `Boss` agent.

1.  **Spawn a Boss**:
    ```bash
    build new boss
    ```
2.  **Interaction**: You will be prompted to describe your goal. The `Boss` will then initiate the "Grill Me" protocol to vet your requirements.
3.  **Decomposition**: Once vetted, the `Boss` uses the `breakdown` tool to generate a task tree and enqueues the work into the Router.

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
You are shielded from the day-to-day work. The system handles 3-strike deadlocks automatically. You only interact with the `Boss` directly when:
1. You are delegating new work.
2. The `Boss` requires clarification on high-level strategy that they cannot resolve themselves.
3. The project goal is complete and ready for your final sign-off.
