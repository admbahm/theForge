# CKB Parser & Graph Validator Implementation

This document describes the Go production parser, structured validation pipeline, and graph compiler for the Career Knowledge Base (CKB) module.

---

## 1. Selected Markdown Dependency: Goldmark

We selected **Goldmark** (`github.com/yuin/goldmark`) as our core Markdown parser library. Goldmark satisfies the parser contract because:
*   **AST Compliance**: It parses standard Markdown into a strongly typed AST (Abstract Syntax Tree), ensuring structural consistency.
*   **GFM Tables Support**: It natively implements GFM (GitHub Flavored Markdown) table nodes via `extension.Table`, which allows parsing metadata tables accurately rather than relying on brittle, error-prone line splits or regexes.
*   **Safety Limits**: Provides a standard scanner that parses blocks sequentially, making it easy to enforce limits on depth and file sizes.

---

## 2. Parsing Pipeline Sequence

Conforming parsers run sequentially as follows:
1.  **Discovery**: Walk directories recursively, sorting alphabetically by relative path. Ignore documentation/template folders.
2.  **Size & Line Limits check**: Reject files exceeding maximum size (1MB) or line length (10,000 bytes).
3.  **PII Scan**: Pre-scan document text for emails and phone numbers. Redact leaked values inside diagnostic reports.
4.  **Metadata Block parsing**: Locate `extast.Table` node at the top of the AST, verify headers, validate sequence ordering, enforce enums.
5.  **Section parsing**: Collect subsequent headings (`## Heading`), extracting prose and canonicalizing unordered list items to `- <content>` while preserving inline Markdown inside the item.
6.  **Local Node validation**: Ensure ID prefix matches object Type.
7.  **Global Graph assembly**:
    *   Register node in `Objects` map. Emit `CKB-IDENTITY-DUPLICATE-ID` if ID collision occurs.
    *   Resolve comma-separated relationship lists against registered objects.
    *   Enforce relationship target compatibility using the resolved target object's declared `Type`.
    *   Perform cycle, self-loop, and orphaned evidence checks.
8.  **Diagnostic sorting**: Sort diagnostics deterministically.
9.  **Export JSON**: Export sorted JSON with stable keys and collections.

---

## 3. Implemented Safety Limits

The following safety control limits are enforced by default:
*   **Max File Size**: 1MB (`1,048,576` bytes).
*   **Max Line Length**: `10,000` bytes.
*   **Max Metadata Rows**: `50`.
*   **Max Object Count**: `1,000` documents per repository.
*   **Max Relationships**: `100` declared relationship targets per node, counted across all relationship metadata rows after trimming empty comma-separated entries and before duplicate-edge detection or graph resolution.
*   **Max Section Count**: `50`.
*   **Max Heading Depth**: `6` (`###### Heading`).

Synthetic evidence nodes discovered inside evidence catalog tables inherit the parent evidence object's parsed `Last Updated` value. They preserve row-level `Visibility` and `Verification Level`; omitted visibility defaults to `Confidential`, omitted verification defaults to `Unverified`, and invalid enum values emit `CKB-METADATA-INVALID-ENUM` without registering the row as authorized public evidence. The parser never uses wall-clock time for synthetic node metadata.
