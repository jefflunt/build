# Design Document: Modular Node-RED Flows & Subflows

## User Story
**Headline**: Modularize the Node-RED Monolith into Versioned Flows and Subflows with Loop-Safe Syncing  
**Problem Statement**: Storing the entire Node-RED orchestrator as a single flat `workflows/sdlc-orchestrator.json` file makes versioning, tracking, and iterating on individual agents or workflow steps difficult. It creates a monolith that is hard to manage and edit modularly.  
**Objective**: Rename the workflows directory, split the monolithic workflow into modular per-agent subflows and deterministic step subflows, store them in `<name>-<version>.json` format inside sibling directories `flows/` and `subflows/`, and update the sync engine to handle bidirectional syncing, deletions, additions, and renames gracefully.  
**Expected Outcome**: 
- The `workflows/` directory is renamed to `flows/`.
- A sibling directory `subflows/` is created.
- The monolithic orchestrator is split into native subflows (for each agent and workflow step) and high-level calling orchestrators, each stored in `<name>-<version>.json` format.
- The `sync-flows` and `deploy-flow` commands are fully aware of this multi-file structure.
- Synchronization is bidirectionally loop-safe, handles additions and deletions based on a local `.build/sync_state.json` tracker, and maintains format-invariance.

---

## Architecture Overview

### Modular Storage Layout
The local workspace will store flows and subflows in two sibling folders:
* `flows/`: Contains high-level orchestrator flows and global configuration files.
  - `sdlc-orchestrator-v1.json`: The high-level orchestrator calling the agent/step subflows.
  - `global-configs.json` (optional): Holds global nodes (like configuration nodes) that are not tied to any specific tab or subflow.
* `subflows/`: Contains native subflow definitions.
  - `dev-v1.json`: Subflow definition for the Developer Agent.
  - `tester-v1.json`: Subflow definition for the Tester Agent.
  - `boss-v1.json`: Subflow definition for the Boss Agent.
  - `sweep-v1.json`: Subflow definition for the Git Sweep Agent.
  - `lead-dev-v1.json`: Subflow definition for the Lead Developer Agent.
  - `query-work-v1.json`: Subflow definition for querying the database for next work.
  - `move-workflow-v1.json`: Subflow definition for transitioning tasks between states/agents.

### Merging and Partitioning (The Flat/Modular Bridge)
Since Node-RED's API works with a single flat array of all nodes in the environment, our synchronization engine acts as a bridge:

1. **Deploying (Merge)**:
   - Read all `.json` files in `flows/` and `subflows/`.
   - Each file contains a JSON array of nodes.
   - Combine all arrays into a single unified flat array.
   - Deploy this flat array to Node-RED via the API.

2. **Retrieving (Partition)**:
   - Fetch the flat array from Node-RED (via API or the local `flows.json`).
   - Group nodes by their parent root:
     - Nodes with `"type": "tab"` are root flows. All nodes with `"z": tab_id` are grouped into this tab's partition.
     - Nodes with `"type": "subflow"` are root subflow definitions. All nodes with `"z": subflow_id` are grouped into this subflow's partition.
     - Nodes without a `"z"` or `"z": ""` are global config nodes. They are grouped into the global config partition.
   - For each partition, match it to its target file using a tracking database/file.

### Deletions, Additions, and Renames (State Tracker)
To handle additions, deletions, and renames without guessing, the sync engine will maintain a light, local JSON state tracking file at `.build/sync_state.json`.
The schema for `.build/sync_state.json` will be:
```json
{
  "last_sync_time": "2026-06-13T12:00:00Z",
  "files": {
    "flows/sdlc-orchestrator-v1.json": {
      "node_red_id": "build-orchestrator-tab",
      "type": "tab",
      "last_known_hash": "...",
      "last_known_mtime": "..."
    },
    "subflows/dev-v1.json": {
      "node_red_id": "subflow:dev-agent",
      "type": "subflow",
      "last_known_hash": "...",
      "last_known_mtime": "..."
    }
  }
}
```

#### Sync State Machine & Decision Tree:
* **Addition Local**: A file exists in `flows/` or `subflows/` but not in `sync_state.json`.
  - It is treated as a new local flow. It is merged, deployed, and added to `sync_state.json`.
* **Addition Remote**: A tab/subflow is in the Node-RED flat array but not in `sync_state.json`.
  - It is treated as a new remote flow. It is saved locally to a `<slugified>-v1.json` file under the appropriate folder, and added to `sync_state.json`.
* **Deletion Local**: A file is tracked in `sync_state.json` but is missing from disk.
  - This means the user deleted the local file. The corresponding flow/subflow partition is removed from Node-RED, and deleted from `sync_state.json`.
* **Deletion Remote**: A tracked file exists on disk, but its corresponding partition is missing from Node-RED.
  - This means the user deleted it in the Node-RED editor. The corresponding local file is deleted from disk, and removed from `sync_state.json`.
* **Modification Local**: The file exists, but its current hash differs from `last_known_hash`, and its mtime is newer.
  - It is deployed to Node-RED.
* **Modification Remote**: The partition exists, but its current remote hash differs from `last_known_hash`, and Node-RED's flows file is newer.
  - It is saved to the local file, overwriting the content.

This guarantees completely robust, predictable, and loop-safe synchronization!

---

## Implementation Backlog

### Pending

### Current

### Completed
- **Task 1: Update Configuration and Core Types**:
  - Update default references to `workflows/` inside code and tests, replacing with `flows/` and adding references to `subflows/`.
- **Task 2: Define and Implement Sync State Tracker and Partitioning Engine**:
  - Implemented loading, tracking, and serialization of local `.build/sync_state.json`.
  - Implemented `PartitionNodes` grouping logic in `internal/syncflow/` separating tabs, subflow definitions, and config nodes.
- **Task 3: Refactor and Split the Monolith**:
  - Renamed `workflows/` to `flows/` and created sibling `subflows/`.
  - Split monolith into `flows/sdlc-orchestrator-v1.json` and 7 subflows under `subflows/` according to the `<name>-<version>.json` convention.
  - Seeded initial tracking metadata in `.build/sync_state.json`.
- **Task 4: Implement Merged Deploy and Partitioned Retrieve Subcommands**:
  - Redesigned `deploy-flow` to aggregate, merge, and deploy all modular files automatically.
  - Implemented the complete 6-state bidirectional sync state machine (Local Add, Remote Add, Local Delete, Remote Delete, Local Mod, Remote Mod) in `sync-flows` using `.build/sync_state.json`.
- **Task 5: Implement Comprehensive Unit and Integration Tests**:
  - Added thorough tests proving loop safety, semantic invariance, and correct state transition routing for deletions and additions.

---

## Checklist & TDD Requirements
* All files in `flows/` and `subflows/` MUST follow the `<name>-<version>.json` pattern.
* Tests must run and pass successfully at every step.
* Tests must prove that adding a local file deploys it, deleting a local file deletes it from remote, and deleting a remote flow deletes it locally.
* Short-circuiting must still prevent any write operations when everything is in sync.
