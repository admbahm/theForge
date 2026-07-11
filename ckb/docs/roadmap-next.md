# CKB Development Roadmap

This document outlines the next engineering phases to transition the Career Knowledge Base (CKB) from a design specification to a production-ready subsystem.

---

## Phase 1: Consolidation & Schema Hardening

Focuses on resolving design weaknesses identified in the architecture review (simplifying redundant documents, stabilizing referencing, and updating templates/examples).

### 1. Document Consolidation
*   **Action**: Merge `publications.md` and `speaking.md` into `contributions.md`; merge `training.md` and `certifications.md` into `credentials.md`; remove `portfolio.md`.
*   **Priority**: High
*   **Difficulty**: Easy
*   **Engineering Value**: High (Eliminates redundant models and synchronizations)
*   **AI Value**: Medium (Fewer document schemas to parse)
*   **User Value**: High (Reduces folder clutter and maintenance overhead)
*   **Risk**: Low
*   **Dependencies**: None

### 2. Standardize ID Relational Referencing
*   **Action**: Replace markdown header anchors with explicit, table-declared string IDs for relational mappings across files.
*   **Priority**: Critical
*   **Difficulty**: Medium
*   **Engineering Value**: Critical (Prevents broken links during user updates)
*   **AI Value**: High (Enables deterministic graph traversal)
*   **User Value**: Low (Changes how links are written but increases robustness)
*   **Risk**: Low
*   **Dependencies**: Document Consolidation

---

## Phase 2: Parser Development & Graph Construction

Focuses on building the software interface to parse Markdown tables, validate schemas, and compile the CKB into an in-memory graph representation.

### 1. Go Markdown Table Parser
*   **Action**: Develop a native Go library to read CKB files, parse the leading metadata tables, and unmarshal them into structured Go structs.
*   **Priority**: High
*   **Difficulty**: Medium
*   **Engineering Value**: High (Enables standard database operations on markdown)
*   **AI Value**: Low (Shields LLM from raw markdown parsing issues)
*   **User Value**: Low
*   **Risk**: Medium (Requires handling diverse markdown editor formatting variations)
*   **Dependencies**: Phase 1 Schema Hardening

### 2. Graph Link Validator & Integrity Tests
*   **Action**: Integrate reference checks into `validation_test.go` to assert that every linked ID (experiences, projects, evidence) exists and is resolved.
*   **Priority**: Critical
*   **Difficulty**: Medium
*   **Engineering Value**: High (Catches broken relationships at build time)
*   **AI Value**: Critical (Ensures the context graph fed to the LLM is intact)
*   **User Value**: High (Alerts user immediately of typos in their vault)
*   **Risk**: Low
*   **Dependencies**: Go Markdown Table Parser

---

## Phase 3: Integration & Generation Tools

Focuses on connecting the parsed CKB database to downstream Forge artifact generators.

### 1. JSON Export & JSON Schema Publication
*   **Action**: Build a CLI command (`theforge ckb export --format=json`) to dump the resolved graph into a standard JSON payload and publish the schema spec.
*   **Priority**: Medium
*   **Difficulty**: Easy
*   **Engineering Value**: High (Decouples Go code from downstream engines in Python or JS)
*   **AI Value**: High (Standardizes model payloads)
*   **User Value**: Medium (Allows exporting data to external web portfolios)
*   **Risk**: Low
*   **Dependencies**: Phase 2 Parser Development

### 2. Evidence-Grounded Resume Engine
*   **Action**: Implement the first downstream generator that consumes the parsed CKB graph, matches requirements against a job posting, and exports a LaTeX or PDF resume.
*   **Priority**: Critical
*   **Difficulty**: Hard
*   **Engineering Value**: High
*   **AI Value**: High (Restricts generated content strictly to trace-linked evidence)
*   **User Value**: Critical (First end-to-end artifact generation workflow)
*   **Risk**: High (Requires careful prompt styling and template formatting)
*   **Dependencies**: Go Markdown Table Parser

---

## Future: Scale & Ecosystem Expansion

Long-term goals for OCKB.

### 1. Obsidian Plugin Development
*   **Action**: Create a community Obsidian plugin providing auto-completion for entity IDs, visual relationship graphs, and metadata form fields.
*   **Priority**: Medium
*   **Difficulty**: Hard
*   **Engineering Value**: Low
*   **AI Value**: Low
*   **User Value**: High (Makes managing CKB accessible to non-technical users)
*   **Risk**: Medium (Requires TypeScript/Obsidian API maintenance)
*   **Dependencies**: JSON Schema publication

### 2. Graph/Vector Database Sync
*   **Action**: Implement direct synchronization pipelines to push parsed CKB nodes into Neo4j and Qdrant.
*   **Priority**: Low
*   **Difficulty**: Hard
*   **Engineering Value**: Medium
*   **AI Value**: High (Allows complex sub-graph matching and vector semantic search)
*   **User Value**: Low
*   **Risk**: Medium
*   **Dependencies**: Phase 3 JSON Export
