# CKB Phase 2 Architecture Decision Log (ADR)

This document captures the key architectural and design decisions finalized during the Phase 2 implementation.

---

## ADR-001: Selected Markdown AST Parser
*   **Context**: We needed a parser that could extract structural components (GFM Tables and Headers) without relying on fragile regex.
*   **Decision**: Selected the **Goldmark** (`github.com/yuin/goldmark`) package with the `extension.Table` compiler.
*   **Consequences**: The parser is highly compliant with CommonMark/GFM syntax and builds an abstract syntax tree, enabling clean walk operations.

## ADR-002: Stable Diagnostic Codes
*   **Context**: Phase 1 verification tests were coupled to english error strings, causing potential fragility when wording was tweaked.
*   **Decision**: Transitioned testing and compiler contracts to stable, uppercase codes (e.g. `CKB-IDENTITY-INVALID-ID`).
*   **Consequences**: Wording, formatting, and row locations can be modified without breaking test compliance.

## ADR-003: Deterministic JSON Array Serialization
*   **Context**: Standard Go JSON encoders serialize map properties in randomized order, violating compilation byte-determinism.
*   **Decision**: Structured objects as a list `[]*model.Object` sorted alphabetically by their ID. Relational target lists are sorted alphabetically during export.
*   **Consequences**: Compiling the vault twice yields byte-identical JSON exports.

## ADR-004: Safe PII Diagnostic Redaction
*   **Context**: Privacy leaks must not be echoed in log output, preventing private credentials or phone numbers from appearing in console runs.
*   **Decision**: Standardized diagnostics messages to replace values with `[REDACTED]` (e.g., `contains email address [REDACTED]`).
*   **Consequences**: Safe compiler output regardless of environment privacy levels.
