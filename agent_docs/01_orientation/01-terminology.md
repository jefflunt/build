# 01-Terminology

This document defines the core entities and architectural concepts for the `build` orchestrator.

## 1. The Human & Orchestrator
- **Owner**: Human (id=1), ultimate decision maker/instruction giver.
- **Agent Registry**: Table containing all participants (Owner, Bosses, Devs, Testers).
- **Router**: Deterministic background service; the *only* entity authorized to move state, assign agents, or trigger escalations.

## 2. The Triad
- **Lead (Boss-X)**: Supervisor holding context, validating work, and managing triad/sub-bosses.
- **Dev**: Implements the task.
- **Tester**: Validates work, writes/runs tests, stamps work (pass/fail).

## 3. Operations
- **Boss-X**: Manages one or more Triads. Confirms work meets goal requirements.
- **Router**: Deterministic background service. The *only* entity authorized to move task status or reassign `assignee_id`.
- **CLI Commands**:
    - `build init`: Initialize local `.build/` repo state.
    - `build design`: Spawn a `Designer` agent for a new project/goal.
    - `build ingest <session-id>`: Import breakdown output into the system.
    - `build route start/stop`: Manage the persistent `Router` service.
    - `build route watch`: Live observability dashboard (TUI).

## 4. Hierarchy
- **Task**: Single, recursive entity representing any unit of work, from a top-level Design down to a granular Dev task.
- **Instruction Integrity**: Every task has a `creator_id`. Instructions are immutable by design; only the `creator_id` can modify or escalate them.

