# Plan 02: Enqueueing Build Plans

## Goal
Establish the workflow and mechanics for enqueuing new development plans into the build system.

## 1. Plan Structure
A build plan is a markdown file that describes the project goal and its breakdown. It should follow this structure:
- `# Goal`: A clear, high-level objective.
- `# Tasks`: A hierarchical breakdown of tasks (using Markdown headers).

## 2. Enqueueing Mechanism
The `build enqueue <plan-file>` command is the entry point for the system.

### Workflow:
1. **Creation**: Create a new `.md` plan file.
2. **Breakdown**: The `build` tool invokes the `breakdown` utility to convert the markdown plan into a directory structure of task files.
3. **Ingestion**: `build` ingests this directory structure into the `build.db` SQLite database.
4. **Reconciliation**: The `Router` service detects the new `todo` tasks and begins the automated execution cycle.

## 3. Automation
- All plans are stored in `agent_docs/04_plans/`.
- The system automatically triggers the reconciliation loop upon ingestion.
