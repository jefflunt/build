# Implement the database querying, task modeling, and structured JSON serialization for 'next', 'blocked', and 'stuck' workflow tasks within a new Go package `internal/taskcmd` to enable clean CLI execution.

This task focuses on implementing the core database querying and serialization logic for workflow tasks in a brand-new Go package, `internal/taskcmd`.

The package will support retrieving three distinct task states: 'next' (representing the next actionable todo leaf task, ordered by priority), 'blocked' (any task currently in 'failed' status), and 'stuck' (any task in 'stuck' status). All query results must be serialized into structured JSON format, returning an empty JSON object `{}` or `null` if no matching task is found.

By packaging this logic within `internal/taskcmd`, we separate SQL execution and structured JSON output from the main CLI router, ensuring high modularity and making the core logic easily testable via dedicated unit tests.
