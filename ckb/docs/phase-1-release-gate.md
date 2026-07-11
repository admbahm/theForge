# CKB Phase 1 Release-Gate Review

This document contains the Principal Engineer's release-gate evaluation of Phase 1: Consolidation and Hardening of the Career Knowledge Base (CKB).

---

## 1. Executive Decision

### **Decision**: PASS WITH NON-BLOCKING FINDINGS

**Phase 2 parser development may begin**, provided that the parser implementation adheres strictly to the rules, edge cases, and parsing syntax defined in [ckb/docs/parser-contract.md](./parser-contract.md). All blocker findings have been successfully mitigated during this hardening pass.

---

## 2. Specification Completeness Audit

| Specification Area | Classification | Notes |
| :--- | :--- | :--- |
| **Object Types** | Complete | All 10 object types are defined with explicit prefixes in [schema.md](../schema.md). |
| **Object Identity** | Complete | Standardized on `^[a-z0-9]+:[a-z0-9-]+$` format. |
| **Metadata Grammar** | Complete | Structured Markdown Table matching specified field ordering. |
| **Data Types** | Complete | Formats for dates, floats, lists, and relations are declared. |
| **Enumerations** | Complete | Enums for `Status`, `Verification Level`, `Visibility`, `Lifecycle State` defined. |
| **Relationships** | Complete | Directional, typed links declared in metadata table. |
| **Evidence Semantics** | Complete | Grounding model separating evidence types, visibility, and verifications. |
| **File Discovery** | Complete | Scope, target paths, and exclusions defined. |
| **Schema Versioning** | Complete | Version declared in metadata table; compatibility rules defined. |
| **Error Handling** | Complete | Error taxonomy and severities defined. |
| **Compatibility Expectations**| Complete | Backwards-compatibility rules and deprecation policy documented. |

---

## 3. Determinism Review (Ambiguity Log)

Two independent parser authors must arrive at the same parsed model. The following potential ambiguities have been identified and resolved:

| Affected File / Area | Rule | Interpretation A | Interpretation B | Recommended Canonical Rule | Severity |
| :--- | :--- | :--- | :--- | :--- | :---: |
| Metadata Parsing | Unknown Fields | Ignore unexpected keys in the table | Throw a parsing error | **Reject the file**. Any unexpected key indicates a typo, which must fail validation. | High |
| Metadata Parsing | Whitespace | Preserve spaces inside values | Trim leading/trailing whitespace | **Trim whitespace**. Value strings must be normalized. | Medium |
| Relationships | Duplicate References | Keep duplicate edges in the model | De-duplicate and raise warning | **De-duplicate and raise validation error**. Duplicate links are invalid. | Medium |
| File Discovery | Templates & Examples | Parse them as part of the CKB graph | Exclude them from graph loading | **Exclude them**. Directories `templates/`, `examples/`, `docs/`, `tests/` are excluded. | High |

---

## 4. Documentation/Test Parity Matrix

This matrix maps all material contracts to their documentation and validation checks:

| Contract | Documented In | Enforced By Test | Status | Notes |
| :--- | :--- | :---: | :---: | :--- |
| **ID format** | `schema.md` | `validation_test.go` | **Enforced** | Verifies `^[a-z0-9]+:[a-z0-9-]+$`. |
| **Duplicate IDs** | `schema.md` | `validation_test.go` | **Enforced** | Asserts global uniqueness across all files. |
| **Metadata order** | `metadata.md` | `validation_test.go` | **Enforced** | Strict row ordering validated. |
| **Required fields** | `schema.md` | `validation_test.go` | **Enforced** | All 10 mandatory fields checked. |
| **Object type enums** | `schema.md` | `validation_test.go` | **Enforced** | Validates against the 10 core types. |
| **Date formats** | `metadata.md` | `validation_test.go` | **Enforced** | Matches `YYYY-MM-DD` standard. |
| **Confidence values** | `schema.md` | `validation_test.go` | **Enforced** | Validates float in range $0.00$ to $1.00$. |
| **Verification values**| `schema.md` | `validation_test.go` | **Enforced** | Strict enum checks (e.g. `Unverified`). |
| **Visibility values** | `schema.md` | `validation_test.go` | **Enforced** | `Public`, `Confidential`, `Internal` check. |
| **Relationship types** | `schema.md` | `validation_test.go` | **Enforced** | Checks prefix compatibility (e.g. `Related Projects` must link to `proj:`). |
| **Broken references** | `schema.md` | `validation_test.go` | **Enforced** | Verifies target ID exists. |
| **Orphan evidence** | `schema.md` | `validation_test.go` | **Enforced** | Asserts evidence nodes are linked. |
| **Unexpected fields** | `metadata.md` | `validation_test.go` | **Enforced** | Rejects extra rows in metadata. |
| **PII detection** | `README.md` | `validation_test.go` | **Enforced** | Regex scan for emails and phone numbers. |
| **Schema version** | `schema.md` | `validation_test.go` | **Enforced** | Validates version `1.0`. |
| **Required sections** | `schema.md` | `validation_test.go` | **Enforced** | Heading checks for Profile, Timeline, Exp, Project. |
| **File-discovery rules**| `parser-contract.md`| `validation_test.go` | **Enforced** | Walks directory, skips templates/docs. |

---

## 5. Graph Integrity Review

How the relation models handle specific structural states:

*   **One-to-One / One-to-Many / Many-to-Many Links**: Supported natively via comma-separated list values in metadata fields.
*   **Directional vs Reciprocal Links**: Relationships are directional (e.g. `experience -> project`). Reciprocal back-links (e.g. `project -> experience`) are computed dynamically by the graph builder rather than declared manually, avoiding double-declaration conflicts.
*   **Self-References / Cycles**: Self-references are strictly prohibited and cause a fatal parsing error. Cycles in career graphs are normally impossible (time flows forward) and should be flagged as warnings.
*   **Dangling References (Broken Links)**: Causes a fatal validation error.
*   **Orphaned Nodes**:
    *   *Orphaned Experiences / Projects*: Flagged as **warnings** (they should be linked to the timeline).
    *   *Orphaned Evidence*: Flagged as **validation error** (no unreferenced evidence nodes should clutter the database).
*   **Deleted / Superseded Nodes**: Nodes marked as `Superseded` or `Deprecated` status are parsed but excluded from active generation views.

---

## 6. Evidence & Provenance Review

### Conceptual Representation Checklist:
1.  **One claim supported by multiple evidence nodes**: Supported. The accomplishment or project metadata links to multiple comma-separated `ev:` IDs.
2.  **One evidence node supporting multiple claims**: Supported. Multiple experiences or projects can list the same `ev:` ID in their metadata tables.
3.  **Private evidence whose contents are not committed**: Supported. The metadata references the ID (e.g., `ev:eval-stark-2025`) and the path points to local store (`local://secure/eval-stark-2025.pdf`), keeping the file outside version control while maintaining the link.
4.  **Evidence that is no longer accessible**: Supported. Verification Level can be marked `Disputed` or `Superseded`.
5.  **Conflicting evidence**: Supported. Verification Level marked `Disputed`.
6.  **Disputed evidence**: Supported. Verification Level marked `Disputed`.
7.  **Superseded evidence**: Supported. Verification Level marked `Superseded`.
8.  **Self-attested claims**: Supported. Verification Level marked `Self-Attested`.
9.  **Independently verified claims**: Supported. Verification Level marked `Independently-Verified`.
10. **Claims with no supporting evidence**: Supported. Verification Level marked `Unverified`.

---

## 7. Privacy Audit Report

*   **Committed File Scan**: No email addresses, phone numbers, real names, or actual company/employer history are present. Fictional examples (`Stark Industries`, `Acme Corp`, `Metropolis University`) are used throughout.
*   **Validation Verification**: The unit tests enforce this state using standard regex filters.

---

## 8. File and Directory Semantics

To prevent parser developers from building directory-coupled code:
*   **Type Determination**: The object type is derived **exclusively** from the `Type` field in the metadata table, not from file paths.
*   **Identity Independence**: Object identity is derived **exclusively** from the `ID` metadata field. Filenames and directory paths can be renamed or relocated anywhere under `ckb/` (excluding skipped folders) without breaking the object graph.
*   **Filename Rules**: Filenames are organizational, not functional. No filename conventions are enforced for compilation.
*   **Skipped Directories**: `templates/`, `examples/`, `docs/`, `tests/` are excluded fixtures.

---

## 9. Failure Semantics & Error Taxonomy

Canonical categories for parser failures:

| Error Category | Severity | Description |
| :--- | :--- | :--- |
| **Malformed Markdown** | Fatal | Parse failure of the basic markdown document. |
| **Missing Metadata** | Fatal | File does not start with a markdown table. |
| **Duplicate Metadata Fields** | Fatal | Key listed twice in the metadata table. |
| **Invalid Field Order** | Validation Error | Required fields do not match canonical ordering. |
| **Invalid Object ID** | Fatal | ID does not match `^[a-z0-9]+:[a-z0-9-]+$`. |
| **Duplicate Object ID** | Fatal | Two files declare the same ID. |
| **Unknown Object Type** | Fatal | Type field is not one of the 10 allowed types. |
| **Invalid Enum Value** | Validation Error | Value is not in the allowed enum list. |
| **Invalid Date** | Validation Error | Date does not parse as YYYY-MM-DD. |
| **Invalid Number** | Validation Error | Float/Int does not parse correctly. |
| **Malformed Relationship** | Validation Error | Relation list is malformed. |
| **Broken Reference** | Validation Error | Link points to a non-existent ID. |
| **Orphan Evidence** | Warning | Evidence node has no incoming edges. |
| **Prohibited PII** | Fatal | Matches email/phone PII patterns. |

---

## 10. Final Findings List

| Finding ID | Classification | Description | Affected Files | Action Required | Blocks Phase 2 |
| :--- | :---: | :--- | :--- | :--- | :---: |
| **CKB-001** | Informational | Consolidate speaking and publications. | `speaking.md`, `publications.md` | Merged into `contributions.md`. | No |
| **CKB-002** | Informational | Consolidate training and certifications. | `training.md`, `certifications.md` | Merged into `credentials.md`. | No |
| **CKB-003** | Low | Remove derived views to enforce DRY. | `portfolio.md` | Deleted. | No |
| **CKB-004** | Low | Rename career objective. | `career-objective.md` | Renamed to `profile.md`. | No |
| **CKB-005** | High | Header anchors are fragile links. | All entities | Replaced link format with CKB IDs. | No (Resolved) |
