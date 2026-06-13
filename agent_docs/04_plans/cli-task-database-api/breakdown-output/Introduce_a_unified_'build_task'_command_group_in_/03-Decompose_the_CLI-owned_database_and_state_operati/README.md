# Decompose the CLI-owned database and state operations ('build task') feature into cohesive, actionable Logical Units of Work (LUoW) by grouping CLI routing and argument parsing together with the respective database querying and transactional update logic.

The 'build task' command group is designed to decouple Node-RED from direct SQLite access, ensuring the workflow orchestrator is database-blind. Currently, Node-RED contains raw sqlite3 command executions which are error-prone and risk deadlocking the database. By exposing safe Go endpoints through the CLI, Node-RED only needs to execute subcommands and consume structured JSON outputs.

To ensure we work on complete, actionable, and testable Logical Units of Work (LUoWs), the implementation is split into two logical slices:
1. Querying subcommands ('next', 'blocked', and 'stuck'): Bundles the SQL selection logic, JSON serialization, and the respective command routing/parsing.
2. State mutation subcommand ('update'): Bundles the complex flag parsing (status, agent, comment insertions, audit logging), safe Go transactional executions to perform multi-table updates atomically, and command routing/parsing.
