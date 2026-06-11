# Design: Integrate Agent CLI as Core LLM Router

## User Story
- **Headline**: Support `agent` CLI as the Primary LLM Router
- **Problem Statement**: The current orchestrator implementation in `build` is tied to `opencode` CLI and requires setting up multiple LLM environment variables (`BUILD_LLM_PROVIDER`, `BUILD_LLM_MODEL`). Maintaining provider/model validation and manual model suggestion guides is redundant now that `agent` CLI handles all routing, environment configurations, and adapters transparently.
- **Objective**: Integrate with the external `agent` CLI tool. Decouple `build` from any direct knowledge of LLM environment variables, providers, or models. Remove model-listing suggestions, and shell out directly to `agent` by feeding the entire prompt to its `STDIN`.
- **Expected Outcome**: `build` will run using `agent:<adapter-name>` (supporting optional whitespace after the colon). All environment variable validations are removed. `build` will pipe prompts directly to `agent <adapter-name>` and receive standard output streams.

## Architecture Overview
The system architecture shifts to delegate all LLM execution directly to the `agent` CLI subprocess:

```
                            +-----------------------+
                            |     build/router      |
                            +-----------+-----------+
                                        |
                                        | Prompts on STDIN (Markdown instructions)
                                        v
                            +-----------------------+
                            |       agent CLI       |  (invokes `agent <adapter-name>`)
                            +-----------+-----------+
                                        |
                            +-----------+-----------+
                            |                       |
                            v                       v
                   [ ~/.agent/config.yml ]     [ Target LLM ]
```

### Components Changed
1. **`internal/config/config.go`**:
   - Support `agent_adapter` values formatted as `agent:<adapter-name>` or `agent: <adapter-name>` (where `<adapter-name>` doesn't contain a slash).
   - Clean up `ParseAdapter` logic to recognize `agent` as a special CLI name where the rest of the string is the adapter name, bypassing the slash requirement.
   - Remove any provider and model environment variables/default fallbacks from config loading.

2. **`internal/cli/`**:
   - Define a new `AgentClient` in `internal/cli/agent.go` that implements the `cli.Client` interface.
   - `AgentClient`'s `Run` method will execute the subprocess `agent <adapter-name>` with the entire prompt piped into `STDIN`.
   - Remove the `Models` method or have it return `nil, nil`, and remove it from `cli.Client` if possible. (Wait, let's keep `Models` returning an empty slice or remove it from the interface, let's check how the interface is defined. We will remove it or stub it if still needed, but better to remove from interface to completely deprecate).

3. **`cmd/build/main.go`**:
   - Completely remove `validateLLMConfig()`, suggestions of models, and environment variable initialization/validation (`BUILD_LLM_PROVIDER` and `BUILD_LLM_MODEL`).
   - Simplify `runRouter()` to just load configuration, instantiate the CLI client, initialize database, write PID, and start the Router.

4. **`internal/router/router.go`**:
   - Simplify the Router construction and parameters to not rely on/use LLM provider and model environment variables.
   - Clean up any audit log entries that expected provider/model by passing empty strings or placeholder strings if the DB schema strictly demands them.

## Implementation Backlog

### Pending

### Current

### Completed
- [x] Task 1: Update `internal/config/config.go` and `internal/config/config_test.go` to support `agent` adapter format and remove environment variables.
- [x] Task 2: Implement `AgentClient` in `internal/cli/agent.go`, update `internal/cli/cli.go` and associated test files.
- [x] Task 3: Simplify `cmd/build/main.go` and `cmd/build/main_test.go` to remove environment variable checking and model-listing functions.
- [x] Task 4: Refactor `internal/router/router.go` and `internal/router/router_test.go` to remove reliance on LLM provider/model and verify all tests pass.

## Checklist & TDD Requirements
- **TDD Requirement**: Write failing unit tests first (RED), implement the minimal changes to make them pass (GREEN), and confirm.
- **Robust Parsing**: Support parsing `agent_adapter: "agent:test_opencode"` and `agent_adapter: "agent: test_opencode"` correctly.
- **Subprocess Mocking**: Use custom runner overrides or test doubles to avoid invoking the actual `agent` CLI during unit tests.
- **No Env Vars**: Ensure no code references or validation errors remain for `BUILD_LLM_PROVIDER` or `BUILD_LLM_MODEL`.
