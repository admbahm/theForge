# Metric Semantics Specification

This document defines the Career Knowledge Base (CKB) taxonomy for quantitative metric classifications, distinguishing verified results from targets, estimates, and contextual markers.

---

## 1. Taxonomy of Quantitative Metrics

Every parsed numeric value inside a claim statement is classified into one of the following semantic types:

| Metric Type | Semantic Meaning | Example Context | Status |
| :--- | :--- | :--- | :--- |
| **Result Metric** | A measured, completed outcome demonstrating improvement or scale. | `"Reduced latency by 40%"` | Approved |
| **Target Metric** | A goal or boundary condition that was aimed for but not necessarily hit. | `"Target was 99.9% uptime"` | Target |
| **Estimate** | A projection or hypothetical forecast of potential savings or gains. | `"Could save 20 hours weekly"` | Estimate |
| **Forecast / Plan** | A scheduled future milestone or prospective milestone. | `"Planned 30% increase"` | Estimate |
| **Range / Approx** | An approximate value indicating uncertainty. | `"Improved by 10-15%"` | Ambiguous |
| **Negative Result** | Explicit statement of zero or lacking improvement. | `"No measurable improvement"` | Ambiguous |
| **Contextual Number** | A volume or scale descriptor that is not an outperformance. | `"Responsible for 40 apps"` | Contextual |
| **Team-Size Value** | Count of personnel managed or supported. | `"Managed team of 12"` | Contextual |
| **Budget Value** | Financial capital allocated or managed. | `"Budget of $2 million"` | Contextual |
| **Duration Value** | Temporal lengths or project periods. | `"Project lasted 18 months"` | Contextual |
| **Version / ID** | Software releases or project indexes. | `"Worked on version 2.0"` | Ambiguous |

---

## 2. Lexical vs. Semantic Metric Extraction

*   **Lexical Extraction**: The system uses regular expressions (`rePct`, `reRps`) to identify numeric candidates and unit strings (e.g., `%`, `RPS`, `requests per second`). This process only identifies the presence of a number; it does **NOT** determine its semantic validity.
*   **Semantic Classification**: A sequence of conservative keyword filters and context rules evaluates the statement's prose structure around the number:
    1.  If the statement contains indicators of future potential or projection (`"expected"`, `"could"`, `"forecast"`), it is classified as an **Estimate**.
    2.  If the statement contains goals (`"target"`, `"goal"`), it is classified as a **Target**.
    3.  If the statement represents software versions (`"version"`, `"v2"`), it is flagged as **Ambiguous** and stripped from achievements.
    4.  If the statement contains scale context (`"team"`, `"budget"`, `"months"`, `"supporting"`), it is classified as **Contextual**.
    5.  It is promoted to a **Result Metric (Approved)** only if it is accompanied by an outperformance verb (e.g., `"reduce"`, `"cut"`, `"save"`, `"increase"`, `"improve"`).

---

## 3. Contribution to Relevance Scoring

*   Only **Approved** (unambiguous, outperformance Result Metrics) contribute to the claim being classified as `KindMeasurableResult`.
*   Claims containing only **Target**, **Estimate**, **Contextual**, or **Ambiguous** metrics remain classified as standard `KindAccomplishment`.
*   During relevance scoring, `KindMeasurableResult` claims receive a scoring bonus (typically +10) over standard `KindAccomplishment` claims because they represent verified quantitative outcomes rather than qualitative descriptions.

## 4. Conflict Comparability

Lexically extracted generic percentage metrics use the placeholder identity `Percentage Outperformance`. That placeholder is not a semantic metric key and is not comparable for blocking metric-mismatch conflicts. Blocking metric conflicts require an explicit shared metric identity and unit; two percentages do not conflict merely because both use `%`.
