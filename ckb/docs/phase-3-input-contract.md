# Phase 3 Input Contract — Downstream Consumption Rules

This document establishes the binding rules and schema constraints that downstream artifact generation engines (resume, CV, professional biography, STAR stories, and skills summary engines) must follow when consuming the Career Knowledge Base (CKB).

---

## 1. Core Consumption Policies

All downstream generation engines in The Forge must adhere to the following rules:

1.  **No Raw Markdown Parsing**: Engines must exclusively consume the strongly typed `*model.KnowledgeBase` domain graph. Direct reading, grep-scanning, or raw parsing of `.md` vault files is strictly prohibited.
2.  **No Filename or Directory Inferencing**: File paths and directory structures (e.g. `experience/acme.md`) must be treated as transport locations only. Engines must never infer metadata, roles, employer names, or dates from folder names or filenames. All details must be extracted directly from the compiled object metadata.
3.  **Strict Diagnostics Gate**: Engines must never run or generate documents if the parsed CKB contains any diagnostics with severity `Error` or `Fatal`. Warnings (e.g. orphaned evidence) may be bypassed but must be reported in logs.
4.  **Lifecycle Verification**: Engines must only select objects whose `Lifecycle State` is `Active` or `Completed`. Objects marked as `Draft` or `Archived` must be ignored.
5.  **Visibility Boundary**: Engines must honor visibility rules. For example, if a resume target is "Public", any CKB object or section marked as `Private` or `Internal` must be filtered out.
6.  **Provenance Enforcement (Critical)**: Every bullet point, paragraph, or certification line in a generated artifact must record the source CKB Object ID (`Source Provenance`) that verified the statement. Fact fabrication or statement synthesis without direct CKB references is strictly prohibited.
7.  **Evidence Provenance Mapping**: Any claim involving key metrics (e.g., "reduced latency by 40%") must have a valid path of relationships in the CKB graph leading to at least one `Evidence` object verifying that metric.

---

## 2. Minimum Input Requirements per Generator

To ensure generation engines produce high-fidelity, complete documents, the CKB graph must contain a minimum set of nodes before starting a build:

### One-Page Resume Generator
*   **Profile**: Exactly 1 `Profile` object marked as `Public` and `Active`.
*   **Experience**: Between 2 and 4 `Experience` objects with `Status: Active` and a confidence score `Confidence >= 0.8`.
*   **Education**: At least 1 `Education` object (completed degree).
*   **Evidence**: A minimum of 1 `Evidence` object mapped to each experience node containing key metrics.

### CV (Curriculum Vitae) Generator
*   **Profile**: Exactly 1 `Profile` object.
*   **Experience**: All `Experience` objects (no count limit, chronologically sorted by the engine).
*   **Education**: All `Education` objects.
*   **Credentials**: All active certifications (`Credentials` type).
*   **Contributions**: At least 2 `Contributions` objects (publications, open-source work, speaking roles).

### Professional Biography Generator
*   **Profile**: Exactly 1 `Profile` object.
*   **Experience**: At least 2 `Experience` objects containing descriptive summary body texts.
*   **Accomplishments**: At least 2 `Accomplishments` or `Contributions` objects.

### STAR Story (Situation, Task, Action, Result) Generator
*   **Context Node**: Exactly 1 `Project` or `Experience` object containing distinct markdown body headings for "Challenge/Situation", "Action", and "Result".
*   **Evidence**: At least 2 distinct `Evidence` IDs linked to the context node confirming the outcome/metrics of the "Result" block.

### Skills Summary Generator
*   **Skills Nodes**: At least 1 `Skills` index object.
*   **Proof Links**: The skills node must have outward relationship references linking to at least 2 distinct `Experience` or `Project` objects where the skill was applied in production.
