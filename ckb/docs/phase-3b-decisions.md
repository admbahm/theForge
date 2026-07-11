# Phase 3B Architectural Decisions

This document details key architectural decisions resolved during Phase 3B implementation.

---

## 1. Non-AST Decoupling
To avoid compiler dependencies and complex Markdown node transformations, the rendering layer consumes only the strongly-typed `ArtifactPlan` and `Claim` schemas. This decoupling makes testing much simpler and improves rendering performance.

---

## 2. Structured Manifest Auditing
Rather than appending loose notes or embedding HTML comments, all audit linkages (entry-to-claim IDs, source references, tenses) are recorded inside a structured `ArtifactManifest` block.

---

## 3. Strict Verification & Preservation Lattice
We chose to enforce language strength at compile time using a strict lattice checker. Any upgrade of fact strength (e.g. from `"contributed"` to `"led"`) triggers an immediate warning diagnostic, guaranteeing correctness without requiring manual developer oversight.
