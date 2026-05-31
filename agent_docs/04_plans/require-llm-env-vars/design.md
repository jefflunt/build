# Require LLM Environment Variables for Router

## User Story
- **Headline**: Require LLM configuration environment variables for `build start`
- **Problem Statement**: Currently, `build start` implicitly falls back to a hardcoded Google Gemini model. This can cause silent failures or unintended costs/runs on incorrect models if the user doesn't realize it's running. Users need strict, clear configuration requirements and helpful setup diagnostics.
- **Objective**: Require both `BUILD_LLM_PROVIDER` and `BUILD_LLM_MODEL` to be explicitly set before `build start` runs. If either is missing, halt execution immediately, print a detailed, helpful setup guide listing available models from `opencode`, and exit with code 1.
- **Expected Outcome**:
  Executing `build start` without the required environment variables outputs:
  ```text
  Error: Missing required LLM environment variables.
  
  To run the orchestrator, you must export both BUILD_LLM_PROVIDER and BUILD_LLM_MODEL.
  
  Example:
    export BUILD_LLM_PROVIDER="google"
    export BUILD_LLM_MODEL="gemini-3.5-flash"
  
  Available models on your system (retrieved from opencode):
    - google/gemini-2.5-flash
    - google/gemini-3.5-flash
    ...
  ```
  And exits with code `1`.

## Implementation Backlog

### Pending

### Current

### Completed
- `[TEST-UNIT]` Create `cmd/build/main_test.go` and write tests for `ValidateLLMEnv`.
- `[LOGIC]` Implement `ValidateLLMEnv(provider, model string) error` in `cmd/build/main.go`.
- `[LOGIC]` Implement dynamic model suggestions by invoking `opencode models` with a graceful static fallback if the command fails.
- `[CLI]` Integrate `ValidateLLMEnv` check inside `runRouter()` in `cmd/build/main.go` to exit `1` and output help text on error.
- `[DOCS]` Update `README.md` to remove references to default provider/model and document the new env requirements.

## Architecture Overview

**Data Flow:**
1. User runs `build start`.
2. `cmd/build/main.go` (inside `runRouter`) retrieves `BUILD_LLM_PROVIDER` and `BUILD_LLM_MODEL` from `os.Getenv`.
3. It passes these values to `ValidateLLMEnv(provider, model string) error`.
4. If validation fails:
   - It attempts to run `opencode models` to fetch actual supported model names.
   - If that succeeds, it prints them. If not, it falls back to a list of standard recommended models.
   - It prints instructions on how to set the variables and exits with code `1`.
5. If validation passes, the router continues to run as normal.

## Checklist & TDD Requirements
- `[TEST-UNIT]` Write `TestValidateLLMEnv` in `cmd/build/main_test.go` covering:
  - Both provider and model present (should pass).
  - Missing provider (should fail).
  - Missing model (should fail).
  - Both missing (should fail).
- Ensure all tests run and pass using `go test ./...`.
- Modify `README.md` to clearly state that the environment variables are mandatory and show how to configure them.

## Agent Instructions for Implementation
- Read-Analyze-Explain-Propose-HALT!
- Only edit one file at a time.
- Do not edit a file without a test.
- Prove tests pass before moving to the next file.
