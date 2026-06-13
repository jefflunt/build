# Refactor the 'deploy-flow' and 'sync-flows' subcommands in 'cmd/build/main.go' to support modular workspace layouts in 'flows/' and 'subflows/'. Implement multi-file merging for deployment, node partitioning for retrieval, and state machine transitions (local/remote additions, deletions, and modifications) driven by the '.build/sync_state.json' file to achieve robust, loop-safe bidirectional synchronization.

This task refactors the existing monolithic 'deploy-flow' and 'sync-flows' subcommands in 'cmd/build/main.go' to handle the new modular workspace layout (located in sibling directories 'flows/' and 'subflows/'). It implements the bidirectional sync logic using a local tracking state file '.build/sync_state.json'.

First, 'deploy-flow' will be updated to automatically aggregate and merge all flow and subflow JSON files from disk into a unified flat node array, deploying it directly to the Node-RED server.

Second, 'sync-flows' will be updated to implement the full state machine transition decision tree (Local Addition, Remote Addition, Local Deletion, Remote Deletion, Local Modification, and Remote Modification) comparing local files, the remote Node-RED state, and the local '.build/sync_state.json' tracker. This ensures a loop-safe, robust sync engine.
