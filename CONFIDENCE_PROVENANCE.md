# Evidence Confidence & Provenance for Forge Intelligence

This document describes how confidence is computed and how provenance is tracked throughout The Forge's career intelligence pipeline.

---

## 1. Overall Analysis Confidence

To make job intelligence transparent and trustworthy, every generated analysis contains an overall confidence score and level.

### Confidence Levels
- **High**: $\text{Score} \ge 0.80$
- **Medium**: $0.50 \le \text{Score} < 0.80$
- **Low**: $\text{Score} < 0.50$ (or if the job description is completely missing)

### Scoring Heuristic (Max: 1.0)
The overall confidence score is computed deterministically in Go using the following evidence completeness rules:

1. **Job Description Content (Max: 0.50)**
   - **Full Description Available**: `+0.50` (Description exists, is not truncated, and contains valid HTML structure).
   - **Invalid HTML Structure**: `+0.35` (Description exists, but contains unbalanced tags or malformed HTML syntax).
   - **Truncated Description**: `+0.20` (Description exists, but contains truncation indicators like `... see more` or `[truncated]`).
   - **Missing Description**: `+0.00` (No description provided). If the description is missing, the total score is capped at `0.35` and the level is forced to `Low`.

2. **Technologies / Tech Stack (Max: 0.20)**
   - **Explicitly Listed**: `+0.20` (The scraping ingestion populated one or more technologies in the frontmatter `tech_stack` array).
   - **Missing**: `+0.00`.

3. **Salary / Compensation (Max: 0.15)**
   - **Available**: `+0.15` (Either `salary_min` or `salary_max` is greater than 0).
   - **Conflicting Metadata**: `+0.00` (If `salary_min > salary_max` where both are non-zero. Adds a warning explanation).
   - **Missing**: `+0.00`.

4. **Core Job Metadata (Max: 0.15)**
   - **Job ID Available**: `+0.05` (`job_id` is populated in frontmatter).
   - **Job Title Available**: `+0.05` (`title` is populated in frontmatter).
   - **Location Details Available**: `+0.05` (`location` is populated in frontmatter).

---

## 2. Requirement Provenance

Every requirement extracted from the posting is labeled with its origin. 

### Allowed Provenance Values
- `explicit`: Directly and explicitly stated in the job description text.
- `inferred_company`: Inferred from the company's public domain, products, or regulatory requirements (e.g. medical device company implying IEC 62304).
- `inferred_industry`: Inferred based on industry-standard best practices for the target domain (e.g. CI/CD for SRE roles).
- `inferred_title`: Implied by the job title itself (e.g. "Senior Developer" implying leadership/mentorship).
- `missing`: Required information that is completely absent.
- `user_supplied`: Added or verified manually by the candidate.

---

## 3. Evidence Provenance

The system distinguishes why evidence is requested:
- Required because it was explicitly requested in the posting.
- Requested because a requirement was inferred (e.g. from company domain or industry best practices).

---

## 4. Gap Confidence & Interview Themes

- **Gaps**: Are not stated as absolute truths unless explicitly supported. Rather than claiming a candidate lacks a skill absolutely, the system notes: `⚠ Job requires X. Candidate evidence not available.` or `⚠ Requirement inferred from company domain.`
- **Interview Themes**: Each interview preparation theme is labeled with a confidence level (`High`, `Medium`, `Low`) and a detailed reason explaining the classification (e.g., explicit posting requirement vs. inferred company products).

---

## 5. Frontmatter and UI Integration

The computed confidence block is saved directly in the Markdown file's YAML frontmatter:

```yaml
analysis_confidence:
  score: 0.87
  level: High
  explanation:
    - Full job description available
    - Technologies explicitly listed
    - Compensation details available
    - Job ID available
    - Job title available
    - Location details available
```

Additionally, it is rendered as a visual badge and expandable details block at the top of the `## The Forge Intelligence` section in the Markdown body:

<span title="Score: 0.87. Expand below to see details.">🟢 High Confidence (Score: 0.87)</span>

<details>
<summary>Confidence Reasoning</summary>

- Full job description available
- Technologies explicitly listed
- Compensation details available
- Job ID available
- Job title available
- Location details available
</details>
