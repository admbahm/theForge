# Contributing to The Forge

Thank you for contributing to **The Forge**! This guide details the development practices and clean room standards required to maintain the safety, predictability, and ethical integrity of this project.

---

## 1. Clean Room Extension Standard

To preserve the separation of concerns and keep models isolated from watching or provider concerns, adhere to these architectural boundaries:

1.  **Domain Models first**: Core structures and serialization should remain inside `pkg/models`. Do not import filesystem watching libraries or network clients into `pkg/models`.
2.  **Mocking & Interfaces**: Any new external integration (Ollama, file parsers, output serializers) must be defined via Go interfaces. This ensures the orchestration engine can be fully tested without relying on external system services.
3.  **Atomic Operations**: Filesystem edits must always preserve unknown metadata and file permissions. Always write updates using the atomic temp-and-rename pattern implemented in `pkg/engine/orchestrator.go`.

---

## 2. Adding a New Analysis Module

To extend the processing pipeline with a new analysis module or prompt template:

1.  **Define the Interface**:
    Add or modify interfaces in `pkg/engine/orchestrator.go` (or a new package) to model the new module's inputs and outputs.
2.  **Implement the Logic**:
    Create or extend implementations under `internal/` (e.g., if it interfaces with LLMs, add prompt modifications inside `internal/ollama/client.go`).
3.  **Update Prompt Templates**:
    Prompts must explicitly prioritize:
    *   Ethical evidence-based alignment.
    *   Clear demarcation of transferable skills vs. direct experience.
    *   No fabrication of metrics, titles, or experiences.
4.  **Register the Phase**:
    Hook the transition of states in `Orchestrator.handleFile(path)` to run the new module when moving between states (e.g., `intel-ready` $\rightarrow$ `apply` phase).

---

## 3. Testing Discipline

Before submitting code modifications, execute local validations:

```sh
# Format changed files
gofmt -w <changed-go-files>

# Run verification test suites
go test ./...
go vet ./...
```

### Safety Rules for Tests:
*   **No Real Vault Edits**: Never write tests that access or write to your personal Obsidian vault.
*   **Use Temp Directories**: Use Go's `t.TempDir()` to construct temporary directories for filesystem watcher and file operations tests.
*   **Deterministic Fakes**: Do not perform real HTTP calls to local or external AI services during unit tests. Use mocks or fakes that implement the `IntelGenerator` interface.
