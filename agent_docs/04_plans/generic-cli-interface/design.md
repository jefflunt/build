# Design: Generic CLI Code Interface

## User Story
- **Headline**: Support multiple LLM CLIs through a generic, unified interface.
- **Problem Statement**: The orchestrator in `build` is currently hardcoded to use the `opencode` CLI. Users cannot use alternative LLM providers or orchestration tools such as `copilot` CLI or Google's `antigravity` CLI. Furthermore, LLM configurations are managed solely via environment variables (`BUILD_LLM_PROVIDER` and `BUILD_LLM_MODEL`), which is difficult to manage across multiple projects.
- **Objective**: Introduce a unified, generic CLI interface configured via `~/.build/config.yml`. Decouple the orchestrator logic from the specific LLM executable. Implement a hardcoded driver pattern where the configuration string specifies the CLI name, provider, and model in an easy-to-parse format (e.g., `cli: <name>:<provider>/<model>`).
- **Expected Outcome**: The user can configure the active CLI and model in `~/.build/config.yml`. The router process runs seamlessly with `opencode` using the new abstraction, and future developers can easily add drivers for `copilot` and `antigravity` by implementing the simple interface.

## Architecture Overview
We will introduce a new interface package `internal/cli` containing the interface definition and a factory to instantiate the appropriate CLI driver.

### The CLI Interface
```go
package cli

import "context"

type Client interface {
	// RunSession executes the CLI session for the given agent and instruction payload
	RunSession(ctx context.Context, agentName string, instructions string) error
	
	// GetValidModels queries the CLI (or fallback) to retrieve supported models
	GetValidModels(ctx context.Context) ([]string, error)
	
	// GetProvider returns the provider name
	GetProvider() string
	
	// GetModel returns the model name
	GetModel() string
	
	// GetCLIName returns the CLI executable name
	GetCLIName() string
}
```

### Configuration Parsing
We will parse `~/.build/config.yml` for configuration. The file will contain:
```yaml
cli: "opencode:google/gemini-3.5-flash"
```
We will parse this line, split it into:
- CLI name: `opencode`
- Provider: `google`
- Model: `gemini-3.5-flash`

If the configuration file is missing, we will halt with a helpful instruction on how to set up `~/.build/config.yml`.

### High-Level Flow
1. At startup (`build start`), the router reads and parses `~/.build/config.yml`.
2. Based on the parsed `cli` value, it resolves the executable and instantiates the correct `Client` implementation (initially the `opencode` driver).
3. The `Client` is injected into the `Router`.
4. Wherever the router executed `opencode`, it now calls `cliClient.RunSession(...)` or `cliClient.GetValidModels(...)`.

## Implementation Backlog

### Pending
- `[CONFIG]` Implement `~/.build/config.yml` parser that reads the user's home directory configuration and parses the `cli: <name>:<provider>/<model>` format.
- `[INTERFACE]` Define the `cli.Client` interface and the initial `opencode` driver implementing this interface.
- `[INTEGRATION]` Refactor `cmd/build/main.go` and `internal/router/router.go` to use the new `cli.Client` abstraction instead of hardcoded environment variables and `opencode` exec calls.
- `[VERIFICATION]` Ensure all existing tests pass and verify the behavior using mock interfaces or a test driver.

### Current
- `[DESIGN]` Align and approve design file.

### Completed
- None

## Checklist & TDD Requirements
1. **Config Parser Unit Tests**: Write unit tests to verify that `~/.build/config.yml` is parsed correctly under different scenarios (missing file, invalid format, correct format).
2. **Interface Driver Unit Tests**: Write unit tests for the `opencode` driver to ensure it executes the command with correct arguments.
3. **Integration Verification**: Modify existing main test suite/mocks to verify `build start` validates the configuration correctly.
