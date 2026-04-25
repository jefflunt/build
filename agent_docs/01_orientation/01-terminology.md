# 01-Terminology

This document defines the core entities and architectural concepts for the `build` orchestrator.

## 1. The Human & Orchestrator
- **Owner**: The human driving the project. Ultimate decision maker.
- **Deputy**: Manages overall project satisfaction by delegating to Bosses, signing off on work for the Owner.
- **Router**: The deterministic background engine that monitors state and manages task assignment/routing.

## 2. The Triad
- **Lead**: Holds context, verifies work, ensures intent is met.
- **Dev**: Implements the task.
- **Tester**: Writes/runs tests, runs linting, stamps work (pass/fail), and manages merges.

## 3. Operations
- **Boss**: Manages one or more Triads. Confirms work meets goal requirements.
- **Router**: The deterministic background engine that monitors state, manages escalations, enqueues maintenance tasks (cleaning/docs), and assigns tasks.

## 4. Hierarchy
- **Goal** (Owner/Deputy) -> **Epic** (Deputy/Boss) -> **Issue** (Boss/Lead) -> **Task** (Lead/Dev/Tester).
