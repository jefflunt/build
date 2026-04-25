# 02-Handoff-Protocol

This document defines the deterministic rules for task management, agent interaction, and escalations within the `build` orchestrator.

## 1. Instruction Chain Integrity
Instructions flow downward (`Owner` -> `Deputy` -> `Boss` -> `Lead` -> `Builder/Verifier`). 
- **Integrity Rule**: An agent receiving instructions *must* act on them, but cannot modify them.
- **Clarification**: If instructions are unclear, the agent must ask for clarification *upward* in the chain.

## 2. The Escalation Hierarchy
When collaborators (Builder + Verifier) cannot resolve a task, the issue escalates automatically via the `Router`.

| Level | Actors | Max Iterations |
| :--- | :--- | :--- |
| **Operational** | Builder ↔ Verifier | 3 (6 touch points) |
| **Lead Review** | Lead ↔ Builder | 3 |
| **Management** | Lead ↔ Boss | 3 |
| **Strategic** | Boss ↔ Deputy | 3 |
| **Final** | Deputy ↔ Owner | Human intervention |

## 3. The Router Engine
The `Router` is the persistent, deterministic engine driving the system.
- **Responsibility**: Monitors issue `status.json` and `audit.log` for staleness, touch-counts, and completion status.
- **Task Assignment**: The *only* entity authorized to assign or reassign tasks based on the current state machine.
- **Operational Model**: A background service (start/stop/check via PID) that acts as the factory engine.
