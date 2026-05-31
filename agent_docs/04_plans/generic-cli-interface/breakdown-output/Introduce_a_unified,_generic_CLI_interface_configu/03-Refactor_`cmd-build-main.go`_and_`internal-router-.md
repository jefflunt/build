# Refactor `cmd/build/main.go` and `internal/router/router.go` to instantiate the `cli.Client` via the parsed configuration and replace all hardcoded `opencode` executions with the new interface methods.

This task involves modifying the application's entry point (`cmd/build/main.go`) and the core routing logic (`internal/router/router.go`) to utilize the newly introduced `cli.Client` interface. The goal is to decouple the system from the hardcoded `opencode` executable.

First, `cmd/build/main.go` must be updated to parse `~/.build/config.yml`, read the `agent_adapter` configuration, and instantiate the appropriate `cli.Client` implementation (such as the `opencode` driver).

Second, this instantiated client needs to be passed down to the router. `internal/router/router.go` should be refactored to accept the `cli.Client` interface and replace all direct `exec.Command("opencode", ...)` calls with the corresponding methods defined on the interface. This completes the decoupling process, ensuring the orchestration logic remains agnostic to the underlying LLM CLI tool.
