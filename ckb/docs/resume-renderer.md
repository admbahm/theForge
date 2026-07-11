# Resume Renderer Specification

The Resume renderer compiles a professional resume format from selected claims.

---

## 1. Professional Chronology and Groups
*   Roles are grouped under their parent organizations to represent promotion history (rather than repeating organization blocks for consecutive roles).
*   Role and achievements are sorted in reverse chronological order.
*   Formatting respects configured date styles (e.g. `YearOnly` vs. `MonthYear` vs. `Present` limits).

---

## 2. Technical and Professional Skill Merges
*   Individual skill assertions are merged into a single grouped entry per category using clean Oxford lists.
*   Calculated durations are omitted unless explicitly configured, avoiding overlapping timeline inflation.

---

## 3. Strict Budget Invariants
*   The renderer enforces strict budget configuration bounds:
    - Maximum number of roles.
    - Maximum number of achievements per role.
    - Maximum number of education records.
*   If selected claims exceed configured budget thresholds, a `CKB-RENDER-BUDGET-EXCEEDED` warning diagnostic is emitted, and excess entries are dropped.
