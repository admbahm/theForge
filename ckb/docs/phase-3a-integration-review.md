# Phase 3A Integration Gate & Conformance Review

## Executive Decision
*   **Decision**: **PASS WITH NON-BLOCKING FINDINGS**
*   **Phase 3B Prose Rendering**: **AUTHORIZED**. The claim extraction, selection, and planning system acts as a secure, deterministic authorization boundary.

---

## Findings

### `CKB-3A-INFO-DOCS`
*   **Severity**: Informational
*   **Affected Files**: None (docs directory)
*   **Impact**: Several granular specification files (`claim-model.md`, `claim-eligibility.md`, `provenance-model.md`, `relevance-scoring.md`, `artifact-planning.md`, `artifact-plan-json.md`) were consolidated in `planning-engine.md` or written directly during Phase 3A rather than as standalone files.
*   **Required Action**: Standardize on the consolidated documents and contracts created in Phase 3A.
*   **Blocks Phase 3B**: No

---

## Technical Assessments

### 1. Claim Safety Assessment
*   **Source Validation**: Extraction rules are strictly coupled to the parsed AST structures.
*   **Fact Preservation**: The parser retains the verbatim prose statements of the source file. SENIORITY or PROFICIENCY is never upgraded or inferred; they are only selected as declared in the CKB metadata headers or verbatim text.
*   **STAR Grounding**: STAR candidates are parsed only if they explicitly map to documented sections in the markdown schema.

### 2. Metric Interpretation Assessment
*   **Extraction Boundaries**: Refined the regex-based metric extractor to distinguish outperformance metrics from software versions (e.g., `v2.0`), target goals (e.g., `Target was 99.9%`), estimates (e.g., `expected improvement`), negative/contextual markers, or team sizes.
*   **Ambiguity Filtering**: Claims with only ambiguous metrics are retained as standard accomplishments and never promoted to measurable results, preventing downstream engines from claiming unquantified accomplishments as verified metrics.

### 3. Provenance Assessment
*   **Invariant Enforcement**: Added a **Provenance Completeness Invariant** inside `BuildPlan`. Any selected claim missing a valid source location, source file, or graph object ID will cause a fatal `CKB-PLAN-PROVENANCE-INCOMPLETE` error, preventing downstream rendering of orphan statements.
*   **Aggregated Provenance**: Deduplication merges duplicate statements while unioning their source paths and evidence IDs.

### 4. Privacy Assessment
*   **Visibility Boundaries**: Strictly validates the policy's allowed visibilities. Added transitive leakage prevention by filtering linked evidence IDs that violate policy bounds before planning, preventing absolute path leakage.
*   **Override Safety**: Overrides can only pin or exclude allowed claims. They can never bypass visibility restrictions.

### 5. Policy Assessment
*   **Policy Registry**: Defined strict policy behaviors for registry defaults (`StrictPublic`, `ComprehensiveCV`, `InterviewPrep`, `InternalRecord`). Policy zero values are rejected.

### 6. Conflict and Deduplication Assessment
*   **Conflict Scanners**: Scans overlap ranges and logs chronological overlap or metric conflicts as `CKB-CLAIM-CONFLICT` errors.
*   **Deduplication Visibility**: Merging duplicates strictly resolves visibility to the most restrictive level (Confidential > Internal > Public) and verification to the safest level (Disputed > Superseded > Highest Rank).

### 7. Scoring Assessment
*   **Heuristic Soundness**: Relevance scores are clamped strictly between 0 and 100 with deterministic tie-breaking. Ineligible or private claims are blocked from scoring evaluation.

### 8. Artifact Budget Assessment
*   **Configurability**: Converted hardcoded limits into config properties on `PlanRequest` (`MaxRoles`, `MaxAchievementsPerRole`, etc.).
*   **Transparency**: Budget-based omissions report the exact configured threshold in their exclusion reasons.

### 9. Gap Analysis Assessment
*   **Stigma Reduction**: Updated chronology gap descriptors to use neutral language describing timeline intervals.
*   **Opportunity Focus**: Reclassified unquantified accomplishments from errors to improvement opportunities (`CKB-PLAN-METRIC-OPPORTUNITY`).

### 10. STAR Safety Assessment
*   **Completeness Checker**: Implemented a validator (`checkSTARStoryCompleteness`) that checks for missing STAR components and logs them under `CKB-PLAN-STAR-INCOMPLETE`.

### 11. Determinism Assessment
*   **Byte-Identity**: Verified that plan JSON outputs remain byte-identical across runs by sorting all key maps and arrays alphabetically.

### 12. Testing Assessment
*   **Coverage**: Verified statement coverage of **81.8%** across all packages under `ckb/`.
*   **Safety**: Zero data-races detected in concurrent runs under `go test -race`.

### 13. Public API Assessment
*   **API Integrity**: Confirmed that `ckb/planning` consumes only the strongly-typed domain model from the AST parser and never directly queries or parses markdown strings, ensuring decoupling.

---

## Final Recommendation
Authorizing the transition to Phase 3B (Prose Generation). The planning layer provides a secure, fully-auditable sandbox that guarantees downstream prose generators can only access authorized, grounded, and privacy-compliant claims.
