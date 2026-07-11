# Deterministic Renderer Baseline

This document defines the baseline standards, constraints, and conformance rules for the Career Knowledge Base (CKB) rendering pipeline.

---

## 1. Supported Artifact Types and Formats
*   **Resume**: Chronological layout, promotion grouping, skills categories.
*   **CV**: Subtype-sorted comprehensive portfolio (`professional`, `technical`, `executive`, `academic-adjacent`).
*   **Biography**: Pronoun-adapted narrative of varying lengths (short, medium, long).
*   **STAR Story**: Situation, Task, Action, and Result stories.
*   **Skills Summary**: Skill category bullets with non-overlapping duration calculations.
*   **Supported Formats**: Deterministic JSON, Portable Markdown, and ASCII Plain Text.

---

## 2. Factual Transformation Rules
*   **Verb Tense Adaptations**: Modifies leading action verbs to past/present depending on request (e.g. `Lead` -> `Led`).
*   **Grammatical Joining**: Standard Oxford lists and space sentence connectors.
*   **Metric Formatting**: Safe formatting (e.g. `approximately 18%` -> `~18%`).
*   **Prohibited Transformations**: Factual upgrades (ownership, skill, certainty, scope), inventing metrics, synthesizing unbacked chronological context, or adding motivators.

---

## 3. Security, Privacy, and Output Determinism
*   **Privacy Filtering**: Confidential/internal claims are excluded from public scopes. Private evidence locations, files, or phone numbers are stripped from manifests and logs.
*   **Reproducibility**: Shuffled inputs or process timezones/locales must produce byte-identical files.
*   **Audit Manifest**: Each rendered artifact includes a deterministic `ArtifactManifest` with claim mapping tables and a SHA-256 content digest.

---

## 4. Diagnostic Blocking Behavior
*   Public output rendering MUST return `Artifact = nil` and generate blocking `Error` diagnostics on:
    - Visibility boundaries leakage.
    - Factual strength upgrades in any of the 4 independent dimensions.
    - Unresolved active conflicts.
    - Incomplete STAR story components (under strict `Fail` behavior).
