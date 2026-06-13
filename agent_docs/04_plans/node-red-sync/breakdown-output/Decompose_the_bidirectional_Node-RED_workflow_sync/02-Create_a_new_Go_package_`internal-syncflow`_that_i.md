# Create a new Go package `internal/syncflow` that implements semantic JSON normalization and hashing. It should expose functions to parse arbitrary JSON into generic Go objects, serialize them deterministically with alphabetically sorted map keys and 4-space indentation using Go's standard `json` package, and compute a SHA256 checksum of the normalized output for loop-prevention matching.

This task focuses on implementing a dedicated package `internal/syncflow` that handles semantic JSON normalization and hashing. The primary objective is to eliminate false positives in change detection caused by differences in key ordering or whitespace formatting between local files and the remote Node-RED API.

To achieve this, the package will:
1. Parse raw/arbitrary JSON content into generic Go objects (using `interface{}` or map/slice combinations).
2. Re-serialize these parsed generic objects back to JSON bytes using Go's standard library `encoding/json`. Because Go's standard JSON library sorts map keys lexicographically during marshaling, this step guarantees a deterministic key structure.
3. Format the serialized output using a consistent 4-space indentation to align with the visual and formatting standards of local Git-controlled workflows.
4. Generate a SHA256 checksum of the resulting normalized byte slice. This hash acts as a semantic fingerprint that the synchronization engine can compare to determine if logical changes actually exist, bypassing infinite sync loops.

In accordance with the Test-Last Pipeline rule, task implementation focuses solely on writing the production code under `internal/syncflow`; separate validation and tests will be handled downstream.
