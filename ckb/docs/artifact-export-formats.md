# Artifact Export Formats

This document describes the three output serialization formats implemented in Phase 3B.

---

## 1. Deterministic JSON
*   Exports the entire `Artifact` structure including the nested `ArtifactManifest` audit records.
*   Enforces indentation, alphabetical map ordering, and HTML escaping defaults to maintain bit-identical exports.

---

## 2. Portable Markdown
*   Outputs structured Markdown documents:
    - `# <Title>` (Heading 1)
    - `## <Section Heading>` (Heading 2)
    - `- <Bullet>` (List bullets)
*   Strips trailing whitespace and enforces exactly one terminal newline.
*   Does not leak internal paths or comments.

---

## 3. Plain Text (ASCII-safe)
*   Strips all markdown symbols (`**`, `*`, `` ` ``).
*   Enforces custom section header borders (`=== SECTION ===` and `--- Subheader ---`).
*   Ensures clean lines and standard ASCII spacing.
