# CKB Schema Specification

This document defines the formal grammar, schema, and relationship rules for the Career Knowledge Base (CKB) core module. Future parser modules must validate files strictly against these rules.

---

## 1. Object Types Inventory

Every Career Knowledge Base record maps to one of the following first-class object types. Each type has a unique prefix, predefined purpose, required content headings, and relation constraints.

| Object Type | ID Prefix | File Location | Purpose | Required Headings | Evidence Support | Independent Node |
| :--- | :--- | :--- | :--- | :--- | :---: | :---: |
| **Profile** | `profile` | `profile.md` | Forward-looking career objective and preference constraints. | `## 1. Professional Vision`, `## 2. Core Target Profile`, `## 3. Technology Alignment Priorities`, `## 4. Career Constraints & Non-Negotiables` | No | Yes |
| **Timeline** | `timeline` | `timeline.md` | Chronological backbone of transitions, milestones, and breaks. | `## 1. Timeline Structure`, `## 2. Chronological Log` | No | Yes |
| **Experience** | `exp` | `experience/*.md` | Historical professional employment logs. | `## 1. Role Context`, `## 2. Key Achievements` | Yes | No (Must reference timeline) |
| **Project** | `proj` | `projects/*.md` | Deep systems design, architecture, and deployment logs. | `## 1. Project Specifications`, `## 2. Architecture & Design Decisions`, `## 3. Implementation Details`, `## 4. Outcomes & Metrics` | Yes | Yes |
| **Skill** | `skill` | `skills.md` | Domain-based competency matrix. | `## 1. Skill Matrix by Domain` | Yes | Yes |
| **Accomplishment**| `acc` | `accomplishments.md` | Standalone achievements independent of employers. | `## 1. Standalone Accomplishments` | Yes | Yes |
| **Credential** | `cred` | `credentials.md` | Merged certifications, courses, and workshops. | `## 1. Professional Certifications`, `## 2. Professional Training Log` | Yes | Yes |
| **Contribution** | `contrib` | `contributions.md` | Merged publications, speaking engagements, and papers. | `## 1. Speaking Engagements`, `## 2. Publications & Technical Writing` | Yes | Yes |
| **Reference** | `ref` | `references.md` | Contact-free relational vouchers vouching for candidate. | `## 1. Professional Vouchers (Contact-Free)` | No | Yes |
| **Evidence** | `ev` | `evidence.md` | Foundational verification database. | `## 1. Relational Evidence Catalog` | No | Yes |

---

## 2. Object Identity Standard

To ensure that the CKB graph can be successfully represented in databases (SQLite, PostgreSQL, Neo4j) and exported to JSON without naming collisions, all objects declare a standardized ID.

*   **Format Pattern**: `^[a-z0-9]+:[a-z0-9-]+$`
*   **Structure**: `<id-prefix>:<kebab-case-slug>` (e.g. `exp:stark-devops`, `proj:titan-consolidation`, `ev:git-titan-repo`).
*   **Casing Rules**: All lowercase alphanumeric and hyphens. Spaces, capitals, and symbols (except the single colon separator) are strictly forbidden.
*   **Uniqueness Scope**: Globally unique across the entire CKB repository. No two nodes may share the same ID.
*   **Immutability**: IDs are immutable. Renaming an ID requires running a refactoring/migration script to update all files referencing that ID to prevent broken links.

---

## 3. Canonical Metadata Table

All CKB files must start with a markdown table matching this exact shape and field ordering:

| Field | Required | Allowed Values / Types | Description |
| :--- | :---: | :--- | :--- |
| **Schema Version** | Yes | `1.0` | Semantic version of the CKB schema. |
| **ID** | Yes | String matching `^[a-z0-9]+:[a-z0-9-]+$` | Canonical unique node ID. |
| **Type** | Yes | Matching one of the canonical Object Types | e.g. `Experience`, `Project`, `Evidence`. |
| **Status** | Yes | `Draft`, `Active`, `Deprecated` | Document curation workflow status. |
| **Verification Level**| Yes | `Unverified`, `Self-Attested`, `Artifact-Supported`, `Independently-Verified`, `Disputed`, `Superseded` | Reliability level of the content. |
| **Confidence** | Yes | Float ($0.00$ to $1.00$) | Completeness of metrics and fields. |
| **Visibility** | Yes | `Public`, `Confidential`, `Internal` | Access permissions filtering. |
| **Source** | Yes | String | Origin of the data (e.g. `Self`, `Git log`). |
| **Last Updated** | Yes | ISO 8601 Date (`YYYY-MM-DD`) | Date of last modification. |
| **Lifecycle State** | Yes | `Planned`, `Active`, `Completed`, `Archived` | State of the associated career item. |
| **Related Documents** | No | Comma-separated list of IDs | References to other core files. |
| **Related Experience** | No | Comma-separated list of `exp:` IDs | Edges linking to experience nodes. |
| **Related Projects** | No | Comma-separated list of `proj:` IDs | Edges linking to project nodes. |
| **Related Evidence** | No | Comma-separated list of `ev:` IDs | Edges linking to evidence nodes. |
| **Tags** | No | Comma-separated tags | Keywords for classification. |

---

## 4. Relationship Linking Model

Relational links are declared explicitly in the metadata table of a node, pointing directly to the target node's unique ID. Downstream graph engines resolve these references.

### Supported Directed Edges:
*   `timeline -> experience` (one-to-one, via `Related Documents`)
*   `experience -> project` (one-to-many, via `Related Projects`)
*   `experience -> evidence` (one-to-many, via `Related Evidence`)
*   `project -> evidence` (many-to-many, via `Related Evidence`)
*   `accomplishment -> experience` (many-to-many, via `Related Experience`)
*   `accomplishment -> project` (many-to-many, via `Related Projects`)
*   `accomplishment -> evidence` (many-to-many, via `Related Evidence`)
*   `skill -> project` (many-to-many, via `Related Projects`)
*   `skill -> evidence` (many-to-many, via `Related Evidence`)
*   `credential -> project` (many-to-many, via `Related Projects`)
*   `credential -> evidence` (many-to-many, via `Related Evidence`)
*   `contribution -> project` (many-to-many, via `Related Projects`)
*   `contribution -> evidence` (many-to-many, via `Related Evidence`)
*   `reference -> experience` (many-to-many, via `Related Experience`)

### Invalid Relationships & Broken-Link Policy:
*   An object must never link to itself (no self-loops).
*   Any reference to an ID that is not declared in the CKB is treated as a **Broken Reference** and will cause validation checks to fail.
*   **Duplicate Links**: Listing the same ID twice in a relationship field is invalid.

---

## 5. Verification & Confidence Semantics

### Verification Level Meanings:
1.  **Unverified**: Personal recollection. High risk of error or lack of proof.
2.  **Self-Attested**: Documented detail, but lacks direct artifact links.
3.  **Artifact-Supported**: Backed by a verified link to an artifact you created (e.g. `ev:git-phoenix-repo` containing your source code commits).
4.  **Independently-Verified**: Authenticated by an external authority (e.g., manager reviews, official transcripts, credential verifier link).
5.  **Disputed**: Contradictory performance reviews or metrics. AI must exclude disputed nodes from target outputs.
6.  **Superseded**: Superseded by a newer, more complete record.

### Confidence Definition:
Confidence is a floating-point score between $0.00$ and $1.00$. It represents data completeness, calculated or assigned as follows:
*   **1.00**: All mandatory schema fields filled, complete metrics provided, and backed by `Independently-Verified` evidence.
*   **0.80 - 0.99**: All fields filled, backed by `Artifact-Supported` evidence.
*   **0.50 - 0.79**: Mandatory fields filled, but missing direct evidence.
*   **< 0.50**: Missing key descriptions or metrics. AI generators should refuse compilation of nodes with confidence $< 0.50$.
