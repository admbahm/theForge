# Phase 1 Decision Log - CKB Consolidation & Hardening

This document records the architectural and design decisions resolved during Phase 1: Consolidation and Hardening of the Career Knowledge Base (CKB) module.

---

## 1. Metadata Schema Migration: Markdown Tables vs YAML Frontmatter
*   **Issue**: How to represent file-level metadata without compromising the readability of standard Markdown editors or standard git diff engines.
*   **Decision**: Standardize on structured Markdown Tables at the top of each file instead of YAML frontmatter blocks. Enforce strict key ordering and header alignment.
*   **Rationale**: Markdown tables are fully styled in native Markdown viewers (like Obsidian) and their diff outputs show line-by-line additions cleanly in Git, avoiding the messy parsing failures that often occur when YAML blocks are malformed by users.
*   **Alternatives Considered**: YAML frontmatter (rejected due to git diff noise and validation parsing overhead in simple environments), JSON code blocks inside HTML comments (rejected as too non-human-readable).
*   **Compatibility Impact**: High. Disallows traditional frontmatter parser routines. Requires implementing custom table parser logic in subsequent phases.

---

## 2. Global Object Identity Standard
*   **Issue**: Reconciling ad-hoc header anchors with database foreign key requirements and graph traversal.
*   **Decision**: Mandate a globally unique ID matching `^[a-z0-9]+:[a-z0-9-]+$` in the metadata block of every node. 
*   **Rationale**: Prevents reference breakage when document headings are updated. The prefix maps directly to the object type, which makes parsing fast and prevents cross-type linkage errors (e.g. linking a skill where a project is expected).
*   **Alternatives Considered**: Filename-based routing (rejected because filenames are mutable and duplicate filenames can exist in different subdirectories), UUIDs (rejected as too hard for humans to write and reference manually in text files).
*   **Compatibility Impact**: Breaking. All previous header-anchor relative links are deprecated and replaced with direct ID references.

---

## 3. Monolithic vs Split Collection Files
*   **Issue**: How to organize large directories like experiences, projects, and evidence to prevent git merge conflicts while keeping editing simple.
*   **Decision**: 
    *   Split `experience/` and `projects/` files into single-node documents (one file per employer or project) with one metadata block at the top.
    *   Keep `evidence.md` as a unified catalog table. The validator scans its table rows to register individual evidence IDs.
*   **Rationale**: Individual project and experience files prevent team merge conflicts. Evidence, however, consists of very small, low-churn references that are much easier to read and scan in a single unified table list.
*   **Alternatives Considered**: Splitting evidence into hundreds of `evidence/*.md` files (rejected as creating too much filesystem noise for simple links).
*   **Compatibility Impact**: High. File directory structural changes.
