# CKB Metadata Standard

All files in the Career Knowledge Base (CKB) must specify their metadata using a standardized Markdown table at the absolute top of the file, preceding any headers, paragraphs, or body bytes.

---

## 1. Canonical Table Template

The table must follow this exact format and ordering. Empty optional fields must contain the string `None`.

```markdown
| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | <prefix>:<identifier> |
| **Type** | <Type> |
| **Status** | [Draft / Active / Deprecated] |
| **Verification Level** | [Unverified / Self-Attested / Artifact-Supported / Independently-Verified / Disputed / Superseded] |
| **Confidence** | [0.00 to 1.00] |
| **Visibility** | [Public / Confidential / Internal] |
| **Source** | [Source Name / Detail] |
| **Last Updated** | [YYYY-MM-DD] |
| **Lifecycle State** | [Planned / Active / Completed / Archived] |
| **Related Documents** | [Comma-separated CKB IDs / None] |
| **Related Experience** | [Comma-separated exp: IDs / None] |
| **Related Projects** | [Comma-separated proj: IDs / None] |
| **Related Evidence** | [Comma-separated ev: IDs / None] |
| **Tags** | [Comma-separated tags / None] |
```

---

## 2. Field Constraint Definitions

### Schema Version
*   **Allowed Values**: `1.0` (for Phase 1 specifications).

### ID
*   **Format**: String matching lowercase regex `^[a-z0-9]+:[a-z0-9-]+$`.
*   **Examples**: `exp:stark-devops`, `proj:titan-consolidation`, `ev:git-titan-repo`.

### Date Format
*   **Format**: ISO 8601 extended format: `YYYY-MM-DD`.
*   **Examples**: `2026-07-10`.

### Boolean Values
*   Where booleans are used in custom tables, they must write as literal strings: `true` or `false`.

### List Format
*   Lists inside metadata tables are written as comma-separated values (e.g. `tag-1, tag-2`).

### Relationship Format
*   All relational references are comma-separated values containing target unique IDs (e.g. `ev:cert-aws-sa-pro, ev:cert-gcp-pca`).

---

## 3. Privacy Constraint Enforcement

To safeguard personal information, metadata values must never contain PII (e.g., your real name, personal email, actual work dates, or contact links). Fictional descriptors (such as `User`, `Fictional Representative`, or `Self`) must be used for ownership and source fields.
