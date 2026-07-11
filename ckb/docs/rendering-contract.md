# Rendering Contract

This contract defines the operational boundaries and security constraints of the Career Knowledge Base rendering pipeline.

---

## 1. Input Constraints
*   The renderer accepts only an authorized `ArtifactPlan` and explicit rendering configuration choices (`RenderOptions`).
*   The renderer is strictly forbidden from inspecting raw CKB objects, parsing Markdown files, or querying the filesystem.
*   The renderer must use only evidence IDs already authorized and retained in the plan. It must not rehydrate unresolved or filtered evidence references.

---

## 2. Validation Checks
Before compiling content, the rendering layer validates the plan to enforce invariants. Fatal violations halt execution:
*   `CKB-RENDER-INVALID-PLAN`: Nil plan or plan lacking required schema variables.
*   `CKB-RENDER-UNSUPPORTED-ARTIFACT`: Artifact target type not supported.
*   `CKB-RENDER-MISSING-PROVENANCE`: Selected claim lacks valid source ID or file mapping.
*   `CKB-RENDER-VISIBILITY-VIOLATION`: Confidential claims present in public artifact outputs.
*   `CKB-RENDER-BLOCKING-CONFLICT`: Active conflicts present without explicit human override approvals.

---

## 3. Allowed and Prohibited Adaptations
*   **Allowed**: Tense shifting of leading action verbs, clean list joining, expanding approved tech acronyms, and formatting metrics to compact styles (using tildes for approximate values).
*   **Prohibited**: Changing ownership, adding metrics, resolving conflicts without overrides, or upgrading factual statements.
