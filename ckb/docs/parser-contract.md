# CKB Parser Contract Specification

This document defines the strict syntax and parsing contracts that any conforming Career Knowledge Base (CKB) parser and compiler must implement.

---

## 1. Discovery Rules

*   **Directories Scanned**: Recursively scan the root directory of the CKB repository (excluding skipped folders).
*   **File Extensions**: Parse files ending with the `.md` extension (case-insensitive). All other extensions must be ignored.
*   **Excluded Directories**: The parser must ignore `templates/`, `examples/`, `docs/`, and `tests/` directories, and any hidden directories starting with a dot (e.g. `.git`, `.obsidian`).
*   **Symlinks**: The parser must **not** follow symlinks to prevent circular paths and out-of-bounds reading.
*   **Nested Directory Behavior**: Allowed. Subdirectories under permitted folders (e.g., `experience/nested/`) are scanned recursively.
*   **Deterministic Traversal Order**: Files must be processed in **alphabetical order** by their clean, relative path (e.g., `experience/acme-lead.md` before `experience/stark-devops.md`).

---

## 2. Markdown Grammar Subset

The parser must enforce the following structural limits on the Markdown dialect:

*   **Multiple Metadata Tables**: Rejects the file if multiple metadata tables exist. Only one metadata block table is allowed.
*   **Table Placement**: The metadata table must be the first block of non-empty lines in the file. No titles or paragraphs may precede it.
*   **HTML Support**: The parser must **reject** any files containing raw block HTML tags (e.g., `<div>`, `<iframe>`) to ensure document safety. Inline HTML tags (like `<b>`, `<i>`, `<br>`, `<details>`) are preserved as raw string content.
*   **Code Fences**: Allowed. Fenced blocks (` ```go `, ` ```mermaid `) in the body are preserved verbatim.
*   **Nested Tables**: Rejects files containing nested tables inside metadata cells.
*   **Multiline Cells**: Prohibited. Every row in the metadata table must fit on a single line.
*   **Escaped Pipes**: Prohibited inside metadata cells.
*   **Blank Metadata Values**: Prohibited. If an optional field is empty, it must explicitly contain the string `None`.
*   **Extra/Reordered Columns**: The metadata table must have exactly two columns: `Metadata` and `Value`. Reordering or renaming headers, or adding third columns, causes a fatal validation error.

---

## 3. Metadata Grammar

*   **Exact Table Headers**: `| Metadata | Value |` followed by a standard separator row `| :--- | :--- |` or `| --- | --- |`.
*   **Canonical Field Order**: Strict row ordering must be enforced matching:
    1.  `**Schema Version**`
    2.  `**ID**`
    3.  `**Type**`
    4.  `**Status**`
    5.  `**Verification Level**`
    6.  `**Confidence**`
    7.  `**Visibility**`
    8.  `**Source**`
    9.  `**Last Updated**`
    10. `**Lifecycle State**`
    11. Optional relation/tag fields: `Related Documents`, `Related Experience`, `Related Projects`, `Related Evidence`, `Tags`.
*   **Whitespace Normalization**: Trim leading and trailing spaces from keys and values.
*   **Case Sensitivity**: Both keys and enum values are case-sensitive. E.g. `Type` must match exactly.
*   **Duplicate / Unknown Fields**: Rejects the file if duplicate keys or unknown keys are declared in the table.

---

## 4. Object Identity

*   **ID Grammar**: Match `^[a-z0-9]+:[a-z0-9-]+$`.
*   **Prefix Rules**: The prefix must map to the type:
    *   `Profile` -> `profile`
    *   `Timeline` -> `timeline`
    *   `Experience` -> `exp`
    *   `Project` -> `proj`
    *   `Skill` -> `skill`
    *   `Accomplishment` -> `acc`
    *   `Credential` -> `cred`
    *   `Contribution` -> `contrib`
    *   `Reference` -> `ref`
    *   `Evidence` -> `ev`
    *   `Education` -> `edu`
*   **Casing**: Lowercase alphanumeric and hyphens.
*   **Uniqueness**: Asserted globally across the scanned vault.
*   **Path/Filename Independence**: The node identity is independent of the filename and its parent folder. Relocating `exp:stark-devops` to a different directory does not affect the logical graph.

---

## 5. Relationship Grammar

*   **Syntax**: Declared as a comma-separated list of target IDs (e.g. `ev:cert-aws, ev:cert-gcp`).
*   **Direction**: Directed relationships pointing from source to target.
*   **Cardinality**: Many-to-many unless constrained. E.g., `Related Projects` links to multiple projects.
*   **Fanout Limit**: `MaxRelationshipsNode` counts all declared relationship targets across `Related Documents`, `Related Experience`, `Related Projects`, and `Related Evidence` after trimming empty comma-separated entries and before graph resolution. Duplicate declared targets count toward this limit and are not silently truncated.
*   **Self-Reference / Cycles**: An object cannot link to itself (causes validation error).
*   **Duplicate Edges**: Listing the same target ID twice in the same list is rejected.
*   **Unresolved Reference**: Referencing a target ID that does not exist in the scanned set causes a fatal validation error.

---

## 6. Section Parsing (Markdown Body)

*   **Required Headings**:
    *   `Profile` -> `## 1. Professional Vision`, `## 2. Core Target Profile`, `## 3. Technology Alignment Priorities`, `## 4. Career Constraints & Non-Negotiables`
    *   `Timeline` -> `## 1. Timeline Structure`, `## 2. Chronological Log`
    *   `Experience` -> `## 1. Role Context`, `## 2. Key Achievements`
    *   `Project` -> `## 1. Project Specifications`, `## 2. Architecture & Design Decisions`, `## 3. Implementation Details`, `## 4. Outcomes & Metrics`
*   **Section Order**: Headings must appear in numeric order.
*   **Section Body Preservation**: The parser must preserve prose under each section as a clean string, maintaining line breaks and inline Markdown such as bold text. Unordered list items are preserved semantically and emitted in canonical Markdown bullet form as `- <content>` regardless of whether the source marker was `-`, `*`, or `+`. GFM tables inside a section are preserved as Markdown table lines in `Section.Body` for downstream header-based extraction.

---

## 7. Conceptual Parsed Model (Intermediate Representation)

Conforming parsers should unmarshal CKB inputs into this intermediate data structure:

```json
{
  "schema_version": "1.0",
  "source_file": "relative/path/to/file.md",
  "id": "exp:stark-devops",
  "type": "Experience",
  "metadata": {
    "status": "Active",
    "verification_level": "Independently-Verified",
    "confidence": 0.95,
    "visibility": "Public",
    "source": "Manager Appraisals",
    "last_updated": "2026-07-10",
    "lifecycle_state": "Active",
    "tags": ["kubernetes", "devops"]
  },
  "relationships": {
    "Related Documents": ["timeline:main"],
    "Related Projects": ["proj:titan-consolidation"],
    "Related Evidence": ["ev:eval-stark-2025", "ev:soc2-audit-log"]
  },
  "sections": [
    {
      "heading": "## 1. Role Context",
      "body": "* Role: Principal DevOps Architect\n..."
    }
  ],
  "diagnostics": []
}
```

---

## 8. Validation Pipeline Phases

Conforming parser executions must execute in the following deterministic sequence:

1.  **File Discovery**: Walk directories, gather file list, sort alphabetically.
2.  **Structural Parse**: Read markdown, find metadata table, extract headers and raw keys.
3.  **Local Schema Check**: Validate field order, dates, float ranges, and enums for the single file.
4.  **Section Check**: Assert presence of mandatory headers for the specified Type.
5.  **Global Indexing**: Register ID globally. Throw fatal error if duplicate ID is found.
6.  **Relation Resolution**: Resolve comma-separated IDs against the global ID index. Throw fatal error if any target is missing (broken reference).
7.  **Graph Integrity checks**: Assert no self-loops, validate relationship target compatibility using resolved target object types.
8.  **Orphan / Quality Checks**: Identify unreferenced evidence nodes, calculate diagnostics.
9.  **PII Scan**: Regex scan for unauthorized personal patterns.
10. **Output Output JSON / AST**.

---

## 9. Output Determinism

To guarantee that two independent implementations yield equivalent output, the JSON exporter must:
*   Sort all objects in the top-level list **alphabetically by ID** (e.g. `acc:cloud-savings` before `exp:stark-devops`).
*   Sort all relation arrays **alphabetically by ID** (e.g. `ev:adr-012, ev:eval-stark-2025`).
*   Standardize date formatting to `YYYY-MM-DD`.
*   Synthetic evidence rows inherit a deterministic source-derived `Last Updated` timestamp from their parent evidence catalog object.
*   Synthetic evidence rows preserve the row-level `Visibility` and `Verification Level` values from the evidence catalog. If the catalog table omits `Visibility`, synthetic evidence defaults to `Confidential`. If it omits `Verification Level`, synthetic evidence defaults to `Unverified`. Invalid visibility or verification values emit `CKB-METADATA-INVALID-ENUM` and are not treated as public, independently verified evidence.
*   Represent floats to two decimal places (e.g. `0.95`).
*   Format diagnostics list in order of appearance (by file path, then line number).
