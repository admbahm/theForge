# Claim Combination Rules

This document specifies the rules governing when and how multiple selected claims may be merged into a single entry.

---

## 1. Safety Requirements

Renderers may combine separate claims into one statement only when:
*   Both claims are selected in the active plan.
*   Both belong to the same section and role.
*   Their visibility configurations are compatible.
*   No active conflicts exist between them.
*   The merged statement does not upgrade or inflate the strength of either claim.
*   The combined statement retains the union of all claim IDs in its `ClaimIDs` audit property.

---

## 2. Technical Implementation: Skill Combination

To reduce document length, individual skill assertions are combined into a clean Oxford list:
- Source: `"Utilized technology: Go."` and `"Utilized technology: Redis."`
- Combined: `"Utilized technologies: Go and Redis."`
- The provenance union guarantees that auditing sidecars track both source claims.
