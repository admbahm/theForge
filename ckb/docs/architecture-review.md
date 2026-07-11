# Career Knowledge Base (CKB) - Architecture Review

This document contains a critical architectural review of the Career Knowledge Base (CKB) subsystem.

---

## 1. Executive Summary

The CKB architecture is **highly coherent** in its mission to treat professional history as structured data, prioritizing version control, human-readability, and evidence-grounded generation.

*   **Approve for Production?**: **APPROVE WITH CHANGES**. The design establishes correct core abstractions (e.g. separating accomplishments from employers and anchoring claims to an evidence registry), but it requires hardening. It cannot be deployed to production without implementing automated reference resolution, stable ID namespaces, and schema validation.
*   **Merge the Pull Request?**: Yes, to the development branch. It provides the templates and schema structure necessary to begin building parser tooling.
*   **Publish as Open-Source Specification?**: No. It is currently a markdown-convenient format rather than a formal specification. To publish as the *Open Career Knowledge Base (OCKB)*, it must define a JSON Schema target, a formal grammar, and reference resolution specifications.

---

## 2. Architectural Strengths

*   **Dry-Accomplishments Mapping**: Allowing Accomplishments to exist independently of specific employers solves the problem of cross-functional achievements (e.g., a project that spans multiple consultancies or opensource work).
*   **Git-Native Table Structure**: Using Markdown tables for metadata (instead of YAML frontmatter) makes line-by-line diffs clean and highly readable in standard git diffs.
*   **Predictable AI Grounding**: Anchoring claims to a first-class `evidence.md` file provides a deterministic audit trail, reducing LLM hallucinations to zero by allowing downstream engines to strip any statements not backed by verified evidence nodes.

---

## 3. Architectural Weaknesses & Risks

*   **Fragile Linkage (Brittle Anchors)**: Relational edges are constructed using Markdown links targeting header anchors (e.g., `[Link](../experience/leadership.md#stark-industries-fictional)`). If a user edits a header for clarity (e.g., changing "Stark Industries (Fictional)" to "Stark Industries Corp"), **every link pointing to that anchor breaks silently**.
*   **Lack of Identifier Namespacing**: IDs (like `TL-013`, `ev-git-titan`) are defined ad-hoc. Without namespacing or directory-based ID constraints, collision risks increase as the CKB scales over decades.
*   **Monolithic Evidence Log**: Storing all evidence references in a single `evidence.md` file will lead to git merge conflicts as developers and automated scrapers continuously write evidence artifacts.
*   **Derived Artifact Contamination**: Storing `portfolio.md` inside the source database is an architectural anti-pattern. A portfolio is a *derived, generated view* of experience and projects, not a primary source-of-truth document.

---

## 4. Simplicity & File Consolidation Review

To reduce maintenance overhead and enforce the Single Source of Truth (SSOT) principle, we recommend the following modifications to the directory structure:

| Document / Path | Status | Recommendation & Rationale |
| :--- | :--- | :--- |
| `ckb/portfolio.md` | **Remove** | **Portfolios are derived views**. Keeping this file forces redundant synchronization with `projects/` and `experience/`, violating the DRY principle. |
| `ckb/speaking.md` & `ckb/publications.md` | **Merge** | Combine into a single `ckb/contributions.md` file. Both represent thought leadership and external content delivery, using identical metadata schemas. |
| `ckb/training.md` & `ckb/certifications.md` | **Merge** | Combine into `ckb/credentials.md`. Certifications are simply verified subsets of professional training and coursework. |
| `ckb/references.md` | **Simplify** | Move from a standalone file to a section in `ckb/timeline.md` or a simplified relational list, as it has minimal data fields. |
| `ckb/career-objective.md` | **Rename** | Rename to `ckb/profile.md`. Objective statements change depending on the application; this file should instead capture core candidate constraints, preferences, and long-term career direction. |

---

## 5. Data Model & Scalability Review

*   **Unique Identifiers**: A stable system requires replacing ad-hoc IDs with standardized URIs or UUIDs (e.g., `ckb://experience/stark` or `ckb://evidence/git-titan`).
*   **Graph/JSON Translation**: Transpiling CKB to JSON is straightforward because Markdown tables parse easily into nested key-value objects. However, resolving links requires a multi-pass parser to build the relational graph.
*   **Database Ingestion**:
    *   *Relational (SQL)*: Importing into SQLite/PostgreSQL is clean. Tables map to schemas like `experiences`, `projects`, `evidence`, with a middle join-table for many-to-many relationships (e.g., `project_evidence`).
    *   *Graph DBs*: The model maps perfectly to property graphs (Neo4j/GraphQL), where nodes are entities and links are edges.
    *   *Vector Search*: Because CKB files are logically chunked by sections, indexing them into vector databases (using metadata properties as payload filters) is simple.

---

## 6. AI & Evidence System Review

*   **Hallucination Prevention**: The evidence system is structurally sufficient. By enforcing that any generated resume bullet must trace-link back to a verified evidence ID, we turn resume tailoring into a retrieval-augmented constraint problem rather than a generation task.
*   **Chunking & Embeddings**: Because entities (projects, experiences) are separated by clear Markdown headers (`##`), chunking is deterministic. Standard sentence transformers will embed these clean blocks cleanly without noise.
*   **Evidence Normalization**: To prevent the monolithic merge conflict problem, `evidence.md` should be split into a directory structure `evidence/*.md` or normalized directly within the project/experience logs they support, or stored as a structured JSON/YAML list in a single directory.

---

## 7. Metadata Review

*   **Relationship Formatting**: Currently, relationships are written as raw Markdown links inside table cells. This is difficult to parse programmatically. Relationships should be normalized into a dedicated, comma-separated ID list row (e.g., `Related Projects: proj-titan, proj-phoenix`).
*   **Verification Longevity**: The `Verification Level` and `Confidence` values are highly robust. They will remain useful as the system evolves, enabling automated filtering based on the audit requirement of the target industry (e.g., defense roles requiring high-confidence, verified evidence).

---

## 8. Testing & Validation Review

The current `validation_test.go` verifies basic syntax and PII containment. However, the following critical test gates are missing and must be added:
1.  **Reference Integrity Check**: Verify that every link target (e.g., `#project-titan`) actually exists in the target file.
2.  **Duplicate ID Check**: Ensure that no two entities declare the same ID (e.g. preventing two timeline entries from sharing `TL-001`).
3.  **Orphaned Node Detector**: Identify any projects or accomplishments that are not linked to any experience or timeline item (helping the user maintain clean records).
4.  **Schema Compliance Validator**: Assert that every metadata table contains all mandatory fields with valid enum values.

---

## 9. Forge Integration Architecture

Downstream Forge modules (Resume Engine, CV Generator, Interview Coach) must **never** consume the raw CKB Markdown files directly. Direct consumption creates tight coupling and code duplication.

Instead, we propose implementing a **CKB Intermediate Model (Core API)** layer:

```
┌────────────────────────┐
│   CKB Markdown Files   │
└───────────┬────────────┘
            │
            ▼
┌────────────────────────┐
│     Go Parser CLI      │
└───────────┬────────────┘
            │ Generates
            ▼
┌────────────────────────┐
│   CKB Graph (JSON)     │
└───────────┬────────────┘
            │ Consumes
            ▼
┌────────────────────────┐
│     Core CKB Go API    │
└─────┬────────────┬─────┘
      │            │
      ▼            ▼
┌───────────┐┌───────────┐
│  Resume   ││ Interview │
│  Engine   ││  Coach    │
└───────────┘└───────────┘
```

Downstream engines query the *Core CKB Go API*, which resolves the relations and returns clean Go structs, ensuring data constraints are validated globally.

---

## 10. Versioning & Migration Strategy

*   **Schema Versioning**: Include a `Schema Version` field in the metadata table (e.g., `schema_version: 1.0`).
*   **Migration Tooling**: As schemas change, write Go migration scripts that read the old Markdown tables, map the columns, and write back the updated layout.
*   **Deprecation Policy**: Support major schema versions for 12 months, outputting migration warnings during compilation.

---

## 11. Open Standard Review (Open Career Knowledge Base - OCKB)

To make CKB an open standard:
1.  Publish a formal RFC outlining the schema specification.
2.  Define a JSON Schema target so developers in Python, Node.js, or Rust can write compliant validators.
3.  Design a standard CLI command (`ockb validate ./ckb`) to allow editors to check document health.

---

## 12. Final Recommendation

### **Recommendation: APPROVE WITH CHANGES**

#### Required Modifications before Main Branch Merge:
1.  **Consolidate Documents**: Merge `speaking.md`/`publications.md` into `contributions.md`, merge `training.md`/`certifications.md` into `credentials.md`, and delete `portfolio.md`.
2.  **Stable ID Referencing**: Replace fragile header anchor links with table-level ID fields to establish robust relational mapping.
3.  **Harden Tests**: Extend `validation_test.go` to enforce reference resolution and detect duplicate IDs.
