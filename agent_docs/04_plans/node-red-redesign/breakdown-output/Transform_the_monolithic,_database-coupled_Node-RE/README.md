# Transform the monolithic, database-coupled Node-RED workflow architecture into an on-demand, event-driven Hub-and-Spoke system with zero polling. Modify the Go CLI to extend task models with execution metadata and to trigger Node-RED via HTTP POST on state updates. Redesign the Node-RED orchestrator canvas to manage concurrency locks, poll for tasks using 'build task next', and dispatch work to modular subflows which are fully refactored to perform updates through 'build task update' instead of raw SQL.

This project transforms the existing Node-RED workflow from a heavy 5-second polling mechanism with raw 'sqlite3' shell commands into an event-driven, push-based Hub-and-Spoke architecture.

On the CLI side, we will extend the task modeling to retrieve and expose execution metadata ('approval_attempts' and 'lead_interventions') and implement an HTTP triggering mechanism within 'build start', 'build ingest', and 'build redo' to notify Node-RED of state updates.

On the workflow engine side, Node-RED will expose a '/trigger-build' endpoint, implement a processing lock to prevent concurrent overlaps, query for available work using 'build task next', dispatch tasks to the respective assignee subflows, and refactor all subflows to update task statuses exclusively via 'build task update' instead of direct sqlite3 access.
