# 03-Handoff-Protocol

## 1. Core Principle: Gatekeepers
Progression is earned. Work must pass through "Sign-off Gates" at every level of the hierarchy:
- **Task Gate (Tester)**: Signs off on implementation quality (tests/linting).
- **Issue Gate (Lead)**: Signs off on functional intent and task composition.
- **Epic Gate (Boss)**: Signs off on project milestone progress.
- **Goal Gate (Deputy)**: Signs off on overall project goal completion.

## 2. Bi-directional Flow
- **Downward (Delegation)**: Instructions are passed down and are **immutable**. An agent acts on instructions; they do not edit them.
- **Upward (Feedback/Escalation)**: Work moves up as it is validated. If work fails a gate, it is rejected and returned to the previous agent's `todo` queue.

## 3. The "3-Strike" Escalation Rule
1. If an agent submits work, and a supervisor returns it (status: `todo`, increment `touch_count`), the work is sent back.
2. If the same entity is rejected **3 times** at the same level (e.g., 3 failed test attempts by the `Tester`), the `Router` automatically detects a deadlock.
3. **The Router Escalates**: It automatically reassigns the entity to the next level of management (e.g., the `Lead`).
4. The higher-level supervisor resets the "strike" count but now acts as the supervisor for the entity.

## 4. Router Logic Summary

| Situation | Router Action |
| :--- | :--- |
| **All Children are `done`** | Mark Parent as `todo` and assign to next Supervisor. |
| **Work Rejected (Pass/Fail = Fail)** | Return to Subordinate (e.g., Tester -> Dev) and increment `touch_count`. |
| **`touch_count` == 3** | **Escalate**: Reassign to the Supervisor in the hierarchy. |
| **Instructions Missing** | Agent triggers an `Escalation` request upward. |

## 5. Escalation Path
`Dev/Tester` (Triad) -> `Lead` -> `Boss` -> `Deputy` -> `Owner` (Human).
