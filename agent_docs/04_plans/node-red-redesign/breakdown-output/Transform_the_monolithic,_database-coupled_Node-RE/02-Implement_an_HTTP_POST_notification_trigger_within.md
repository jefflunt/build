# Implement an HTTP POST notification trigger within the Go CLI to notify Node-RED of task state updates, and integrate it into the successful execution paths of the 'start', 'ingest', and 'redo' CLI commands.

This task requires implementing an HTTP POST notification utility in the Go CLI ('cmd/build/main.go') and integrating it into the 'start', 'ingest', and 'redo' command routes. When these commands execute successfully, the CLI will fire an HTTP POST request to the configured Node-RED webhook endpoint, initiating the event-driven processing of tasks with zero polling.

The implementation should parse the target Node-RED URL from the configuration, construct a JSON payload with relevant task execution metadata (such as task ID, current state, or trigger source), and execute the HTTP request. Proper timeout handling and error tolerance must be added to ensure that a failure in the webhook notification does not block or rollback the CLI command's primary database operations.
