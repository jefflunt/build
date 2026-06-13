# Design: Node-RED Hosted SDLC Automation Workflow

## User Story
* **Headline**: Port the `build` Orchestration Workflow to Node-RED
* **Problem Statement**: The current task routing and execution logic is hardcoded inside the Go-based `build` router. While functional, this approach lacks flexibility, makes it difficult to visually inspect or modify the workflow, and prevents users from creating custom automated SDLC flows.
* **Objective**: Convert the hardcoded workflow in `build` into an independent Node-RED-hosted workflow that polls the task database, coordinates the Dev -> Tester -> Test Suite -> Boss agent cycle, and uses the `agent` and `build` CLI tools.
* **Expected Outcome**: A fully functional Node-RED flow (deployed directly to Node-RED) that completely handles task orchestration, ensuring no duplicate tasks are processed concurrently, and allowing the user to view, edit, and experiment with the workflow in their browser.

---

## Implementation Backlog

### Pending
- Task 2: Create the Node-RED flow definition JSON file (`node-red-flow.json`) representing the entire Dev -> Tester -> Test Suite -> Boss workflow.
- Task 3: Implement a mechanism to lock/serialize task execution in Node-RED to prevent concurrent/duplicate builds on subsequent polls.
- Task 4: Deploy the Node-RED flow to the active Node-RED instance configured via `node_red_url` using the Node-RED admin API.
- Task 5: Add integration tests or a demo verification script to prove that the Node-RED flow successfully runs the task tree through Dev, Tester, Tests, and Boss to completion.

### Current

### Completed
- Task 1: Update the config module (`internal/config/config.go`) to strictly require `node_red_url` (e.g. `node_red_url: http://localhost:1880`) in `~/.build/config.yml`. If missing, print a helpful error message explaining how to configure it. Update existing tests and example configurations.

---

## Architecture Overview

### System Boundaries and Interaction
Instead of running a persistent background service in Go (`build start` using `internal/router`), Node-RED becomes the central SDLC automation host. It uses CLI tools (`sqlite3`, `agent`, `git`, and `./.build/test`) to interact with the environment.

```mermaid
graph TD
    subgraph Node-RED (Workflow Engine)
        Trigger[Poll Injector - Every 5s] --> LockCheck{Is Lock Active?}
        LockCheck -- Yes --> Stop[Stop]
        LockCheck -- No --> CheckBlocked{Any Blocked Task?}
        CheckBlocked -- Yes (failed) --> Stop
        CheckBlocked -- No --> CheckStuck{Any Stuck Task?}
        
        CheckStuck -- Yes (stuck) --> SetLockS[Acquire Lock] --> RunLead[Run Lead Agent] --> RunSweepS[Run Sweep Agent] --> ReleaseLockS[Release Lock]
        CheckStuck -- No --> GetTodo[Get Next Todo Task]
        
        GetTodo -- None --> Idle[Stop / Idle]
        GetTodo -- Task Found --> SetLock[Acquire Lock] --> RouteAssignee{AssigneeID?}
        
        RouteAssignee -- 2: Dev --> RunDev[Run Dev Agent] --> RunSweepDev[Run Sweep] --> HandtoTester[Assignee -> 3] --> ReleaseLock
        RouteAssignee -- 3: Tester --> RunTester[Run Tester Agent] --> RunSweepTest[Run Sweep] --> RunTests[Execute ./build/test]
        
        RunTests -- Pass --> HandtoBoss[Assignee -> 4] --> ReleaseLock
        RunTests -- Fail --> IncAttempts[Increment attempts] --> AttemptCheck{attempts >= 3?}
        AttemptCheck -- Yes --> EscLead[Status -> stuck, Assignee -> 5, resets] --> ReleaseLock
        AttemptCheck -- No --> HandtoDev[Assignee -> 2] --> ReleaseLock
        
        RouteAssignee -- 4: Boss --> RunBoss[Run Boss Agent] --> RunSweepBoss[Run Sweep] --> ParseBoss[Parse Boss Decision]
        ParseBoss -- Approved --> MarkDone[Status -> done] --> ReleaseLock
        ParseBoss -- Rejected --> IncAttemptsBoss[Increment attempts] --> AttemptCheckBoss{attempts >= 3?}
        AttemptCheckBoss -- Yes --> EscLeadBoss[Status -> stuck, Assignee -> 5] --> ReleaseLock
        AttemptCheckBoss -- No --> HandtoDevBoss[Assignee -> 2] --> ReleaseLock
        
        ReleaseLock[Release Lock] --> Done[Done]
    end
    
    subgraph Workspace
        DB[(.build/build.db)]
        Files[Workspace Files]
        Git[Git Repository]
    end
    
    sqlite3[sqlite3 CLI]
    agent_cli[agent CLI]
    test_script[./.build/test]
    
    RunDev & RunTester & RunBoss & RunLead & RunSweepDev & RunSweepTest & RunSweepBoss --> agent_cli
    GetTodo & CheckBlocked & CheckStuck & HandtoTester & HandtoBoss & HandtoDev & MarkDone & EscLead --> sqlite3
    RunTests --> test_script
    RunSweepDev & RunSweepTest & RunSweepBoss --> Git
    
    sqlite3 --> DB
    agent_cli --> Files
    test_script --> Files
    Git --> Files
```

---

## Checklist & TDD Requirements
- **Verification of Node-RED deployment**: The Node-RED flow must be loaded into Node-RED without errors, and be visible on `http://localhost:1880/`.
- **Database/Agent compatibility**: The flow must correctly query SQLite database `.build/build.db` and run `agent` with correct stdin content and check execution exit status.
- **Task completion**: The entire flow must be shown to take a raw task (e.g. `T1`) from `todo` to `done` after Dev, Tester, and Boss approval.
