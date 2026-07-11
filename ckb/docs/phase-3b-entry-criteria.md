# Phase 3B Entry Criteria

This document defines the strict quality, safety, and correctness criteria that must be satisfied before starting Phase 3B: Prose and Artifact Generation.

---

## 1. Gateway Checklist & Entry Gates

Before checking out the feature branch for Phase 3B, the following verification checklist must be completely satisfied:

*   [x] **Phase 3A Gate Passes**: The Principal Engineer review has authorized passage with zero blockers.
*   [x] **Zero Blockers / High Findings**: There are no active blocker-level or high-severity findings in the issue tracker.
*   [x] **Contracts Approved**:
    *   [Claim Authorization Contract](claim-authorization-contract.md) is signed off.
    *   [Metric Semantics Specification](metric-semantics.md) is signed off.
*   [x] **Safety Invariants Enforced**:
    *   *Provenance Completeness Invariant* is active in `BuildPlan` (fails plan if claim ID, source ID, or source file is missing).
    *   *Transitive Privacy Protection* is active (linked evidence IDs violating policy boundaries are stripped; absolute path names cleaned).
*   [x] **Graph Safety Verification**:
    *   Conflict checks (`CKB-CLAIM-CONFLICT`) identify date overlaps and title variant mismatches.
    *   Deduplication unioning aggregates source locations and evidence IDs while resolving visibility to the most restrictive level.
*   [x] **Determinism**:
    *   Plan JSON serialization (`ExportPlanJSON`) is byte-identical across traversal order changes.
*   [x] **Dynamic Budgets**:
    *   Selection and page budgeting limits are fully configurable on `PlanRequest` and reported in exclusion logs.
*   [x] **Quality Benchmarks**:
    *   Race detector verification passes cleanly (`go test -race ./...`).
    *   All package tests pass cleanly (`go test ./...`).
    *   Statement coverage is above **80%** across `ckb/...` (`81.8%`).

---

## 2. Pull Request Checklist

Any pull request introducing Phase 3B rendering code must satisfy the following checklist items:

1.  **Strict Plan Adherence**: Generator engine only reads from the generated `ArtifactPlan.SelectedClaims` array; it never queries raw CKB objects directly.
2.  **No Fact Injection**: Verifies that the generator does not invent dates, metrics, roles, or achievements.
3.  **Grammatical Normalization**: Adapts tense and formats metrics in accordance with the [Claim Authorization Contract](claim-authorization-contract.md).
4.  **STAR Completeness Enforcement**: Any STAR story generated must contain all Situation, Task, Action, and Result components. If the planner marked it as incomplete, the generator must reject rendering or render the section with a clear warning placeholder.
5.  **No Absolute Path Exposure**: Verifies that the generator does not write local workspace absolute file paths or confidential metadata to the output resume or document.
