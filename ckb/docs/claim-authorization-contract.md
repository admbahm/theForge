# Claim Authorization Contract

This contract defines the strict boundaries and legal rules under which a downstream rendering engine (e.g., resume compiler, biography generator, or interview preparation helper) may use a claim.

---

## 1. Ground Rules for Claim Usage

A downstream renderer **MUST NOT** use any claim unless all of the following conditions are met:

1.  **Plan Selection**: The claim is explicitly listed in the `selected_claims` array of a valid, generated `ArtifactPlan`.
2.  **Eligibility Clearance**: The claim has resolved to `Eligible` under the requested planning policy.
3.  **Visibility Clearance**: The claim's visibility matches or is less restrictive than the allowed boundary configured in the policy.
4.  **Provenance Completeness**: The claim must be accompanied by complete provenance records, containing a source location path and line number, a source object ID, and verification levels.
5.  **No Blocking Conflicts**: The claim must not be part of any active unresolved conflict (`CKB-CLAIM-CONFLICT`) unless explicitly resolved via an override.
6.  **No Unsupported Inference**: The renderer must represent the claim exactly as stated or perform only explicitly permitted syntactic modifications. It must never perform semantic upgrades.
7.  **Override Compliance**: Any human override pinning, excluding, or re-sectioning the claim must be respected.
8.  **Section Assignment**: The claim must only be placed in the section configured for it under the `section` property of the `PlannedClaim`.
9.  **Evidence Authorization**: Evidence IDs are authorized only when they resolve to a known `Evidence` object whose visibility and state are allowed by the active policy. Unresolved evidence IDs are never considered authorized evidence.

---

## 2. Permitted Transformations

Renderers may adapt the presentation of authorized claims to fit layout styles or grammatical flow, subject to the following rules:

*   **Improve Grammar & Style**: Adjust sentence structure, spelling, or capitalization for polished layout presentation.
*   **Combine Compatible Claims**: Merge multiple related claims if they belong to the same section and role (e.g., combining two skill assertions into a list: `"Utilized Redis. Utilized Envoy."` -> `"Utilized Redis and Envoy"`).
*   **Shorten Wording**: Trim verbose prose to meet layout constraints, provided the core meaning is not lost.
*   **Expand Abbreviations**: Expand standard, approved abbreviations (e.g., `"GCP"` -> `"Google Cloud Platform"`).
*   **Adapt Tense**: Shift past tense to present tense (or vice versa) to maintain consistency across the document section.
*   **Format Metrics**: Format numerical values (e.g., `"99.9% uptime"` to `"99.9% High Availability (HA)"` or mapping RPS figures).
*   **Neutral Transitions**: Insert connecting phrases to smooth narrative flow (e.g., `"Furthermore,"`, `"Specifically,"`).

---

## 3. Forbidden Transformations

Renderers are strictly forbidden from performing any of the following alterations:

*   **Add Facts**: Fabricate dates, employers, certifications, roles, project outcomes, or other career details.
*   **Add/Inflate Metrics**: Invent percentages, dollar amounts, team sizes, throughputs, or incident reductions that are not explicitly present in the selected claim's metrics structure.
*   **Change Ownership**: Convert team accomplishments (e.g., `"Participated in the team migration to GCP"`) into personal ownership (e.g., `"Migrated core infrastructure to GCP"`).
*   **Increase Scope**: Amplify scale or quantity details (e.g., upgrading `"supported 5 applications"` to `"managed the entire enterprise application suite"`).
*   **Increase Seniority**: Alter job titles or roles to imply greater seniority (e.g., rendering `"DevOps Architect"` as `"Principal DevOps Director"`).
*   **Increase Proficiency**: Upgrade skill familiarity level (e.g., rendering `"worked with Kubernetes"` as `"expert-level Kubernetes practitioner"`).
*   **Infer Causality**: Manufacture causal links not explicitly present in the source (e.g., converting `"implemented SRE procedures"` and `"reduced downtime"` into `"implemented SRE procedures which directly reduced downtime"` unless explicitly coupled in the source claim).
*   **Resolve Conflicts**: Arbitrarily select one side of a conflict or hide warnings without human operator intervention.
*   **Fill Missing STAR Components**: Attempt to "fill in the blanks" for incomplete STAR stories. Incomplete stories must remain incomplete or throw errors.
*   **Expose Private Provenance**: Include absolute directory paths, private contact info, private URLs, or sensitive identifiers in any public-facing artifact.
*   **Expose Unresolved Evidence IDs**: Include unresolved, malformed, wrong-type, or policy-restricted evidence identifiers in public plans, manifests, sidecars, Markdown, or text exports.
*   **Query Raw Objects**: Bypassing the plan to query raw knowledge base objects to find additional claims is strictly prohibited. The generated plan is the sole source of truth.
