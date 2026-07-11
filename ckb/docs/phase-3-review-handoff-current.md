# Phase 3 Review Handoff: Current Findings

Branch: `feat/ckb-core-design`

Current state at handoff:

- Existing CKB implementation work is staged.
- No implementation changes for the findings below have been started in this handoff.
- Do not change branches.
- Do not push, merge, rebase, reset, clean, discard work, or commit.
- Preserve all staged and uncommitted changes.
- Make focused corrections only. Do not add new CKB features or redesign the pipeline.

## Required Task

Address the three current PR-style `/review` findings on branch `feat/ckb-core-design`.

## Finding 1: Exclude Inactive Source Objects From Generation

### Current Issue

Claim eligibility currently excludes `Draft` records but may still allow claims from objects whose source metadata is:

- `Status: Deprecated`
- `Status: Superseded`
- inactive under another terminal status
- `Lifecycle State: Archived`
- `Lifecycle State: Planned`
- another non-active lifecycle value prohibited by the generation contract

These claims can pass visibility, verification, and confidence checks and become selected for resumes, CVs, biographies, or other artifacts.

The documented contract states that inactive records may remain parseable and inspectable but must be excluded from active generation views.

### Required Behavior

Eligibility must consider the source object's status and lifecycle, not only fields copied onto the claim where those may be incomplete.

Define one canonical generation-eligibility rule for object state.

For normal public and active-generation policies:

- allow only statuses and lifecycle states explicitly approved for active generation;
- exclude deprecated records;
- exclude superseded records;
- exclude archived records;
- exclude planned records unless an artifact policy explicitly allows planned content;
- exclude drafts;
- retain inactive objects in the parsed CKB and internal audit views where appropriate.

Do not delete inactive objects from the graph.

Do not silently reinterpret inactive records as active.

### Policy Behavior To Document

Document behavior for:

- `StrictPublic`
- `ComprehensiveCV`
- `InterviewPrep`
- `InternalRecord`

A comprehensive CV must not automatically mean "include deprecated or superseded facts."

`InternalRecord` may retain inactive claims for inspection, but they must be labeled and must not become ordinary active-generation selections unless the policy explicitly authorizes that use.

### Diagnostics And Reasons

Add or reuse stable eligibility reason codes such as:

- `CKB-CLAIM-INACTIVE-STATUS`
- `CKB-CLAIM-INACTIVE-LIFECYCLE`

or the repository's established equivalents.

The decision must be machine-readable. Do not rely solely on human-readable messages.

### Regression Tests

Add tests for at least:

1. active status plus active lifecycle is eligible;
2. draft status is excluded;
3. deprecated status is excluded;
4. superseded status is excluded;
5. archived lifecycle is excluded;
6. planned lifecycle is excluded from public generation;
7. inactive source role does not appear in resume output;
8. inactive source role does not appear in CV output unless an explicit policy permits it;
9. inactive claims remain available in internal inspection where documented;
10. active and inactive records with similar content do not deduplicate into an active merged claim;
11. inactive evidence does not authorize active output;
12. repeated planning remains deterministic.

Inspect deduplication and normalization to ensure an active claim cannot inherit inactive provenance or vice versa in a way that bypasses the rule.

## Finding 2: Reject Raw HTML Before Parsing CKB Markdown

### Current Issue

Raw block HTML such as:

```html
<iframe src="..."></iframe>
<div>...</div>
<script>...</script>
```

can reach Goldmark parsing and produce a usable CKB object.

The parser contract states that raw block HTML is prohibited and must produce a fatal diagnostic.

Unsafe or malformed HTML must not survive into:

- section bodies;
- claims;
- plans;
- Markdown exports;
- plain-text exports;
- biographies;
- manifests;
- provenance sidecars.

### Required Behavior

Reject prohibited raw HTML before constructing a valid CKB object.

Use structural parsing where possible.

Do not rely on a narrow regex that only detects a few tag names.

At minimum detect and reject Goldmark raw HTML block nodes and any inline HTML nodes prohibited by the parser contract.

Clarify whether the contract rejects:

- all raw HTML;
- block HTML only;
- inline HTML;
- HTML comments;
- escaped HTML text;
- Markdown code spans containing HTML;
- fenced code blocks containing HTML.

Recommended conservative behavior:

- reject actual raw HTML nodes;
- allow escaped text such as `&lt;div&gt;`;
- allow HTML-looking content inside code spans or fenced code blocks if code fences are allowed by the current contract;
- do not execute or sanitize raw HTML into accepted content;
- fail closed.

### Diagnostic

Add or reuse a stable diagnostic such as:

- `CKB-MARKDOWN-RAW-HTML`

The diagnostic should be fatal for the affected document.

Include:

- source file;
- line and column where available;
- safe description;
- remediation.

Do not echo unsafe HTML payloads unnecessarily.

### Regression Tests

Add tests for:

1. raw `<div>` block is rejected;
2. raw `<iframe>` block is rejected;
3. raw `<script>` block is rejected;
4. inline HTML is rejected if prohibited;
5. HTML comment behavior matches the documented contract;
6. escaped HTML text is accepted;
7. HTML inside an allowed fenced code block follows the contract;
8. rejected HTML file does not create a valid object;
9. rejected HTML cannot appear in claims or artifacts;
10. diagnostic code and severity are exact;
11. no parser panic occurs;
12. repeated parsing produces deterministic diagnostics.

Update `parser-contract.md` and parser implementation documentation only where behavior needs clarification.

## Finding 3: Require Exactly Two Metadata-Table Columns

### Current Issue

The canonical metadata table requires exactly:

```markdown
| Metadata | Value |
```

The parser currently accepts tables with three or more columns because it checks only that at least two cells exist and ignores the rest.

This can hide malformed or ambiguous metadata and allows invalid CKB files to compile.

### Required Behavior

The canonical metadata table must contain exactly two columns.

Enforce:

- exactly two header cells;
- expected normalized header names;
- exactly two cells in every metadata data row;
- no ignored extra cells;
- no missing cells;
- deterministic handling of blank values;
- escaped pipes according to the current table contract.

Do not truncate or silently ignore additional columns.

A malformed metadata table must not create a valid CKB object.

### Diagnostic Behavior

Use stable diagnostics that distinguish where practical:

- `CKB-METADATA-INVALID-COLUMN-COUNT`
- `CKB-METADATA-INVALID-HEADER`
- `CKB-METADATA-MALFORMED-ROW`

Reuse existing codes if they already express these failures precisely.

Diagnostics should identify:

- source file;
- row or line;
- expected column count;
- actual column count.

### Regression Tests

Add tests for:

1. canonical two-column metadata table passes;
2. three-column header fails;
3. two-column header with a three-cell data row fails;
4. one-column metadata table fails;
5. extra empty third column still fails;
6. malformed separator row follows the parser contract;
7. escaped pipe in a `Value` cell does not count as an extra column;
8. extra columns are never silently discarded;
9. malformed metadata does not register an object;
10. diagnostic ordering is deterministic;
11. existing valid fixtures still pass;
12. invalid fixture asserts the exact diagnostic code.

Add or update a golden invalid fixture for extra metadata columns.

## Nearby Inspection

Inspect adjacent code for the same contract gaps:

- claim eligibility using status but not lifecycle;
- deduplication combining active and inactive claims;
- parser acceptance of inline HTML;
- other Markdown tables that silently ignore extra columns;
- evidence catalog tables, skill tables, or relationship tables where exact or bounded column behavior is documented.

Make only directly related corrections.

Do not impose the two-column metadata rule on other table types unless their own contracts require it.

## Validation

Run:

```sh
gofmt -w <changed Go files>
git diff --check
git diff --cached --check
GOCACHE=/private/tmp/theforge-go-cache go test ./...
GOCACHE=/private/tmp/theforge-go-cache go vet ./...
GOCACHE=/private/tmp/theforge-go-cache go test -race \
  -coverprofile=coverage.out \
  -coverpkg=./ckb/... \
  ./ckb/tests/...
```

Run focused tests for:

- inactive source exclusion;
- raw HTML rejection;
- exact metadata-column enforcement.

Do not weaken existing parser, privacy, provenance, determinism, planning, rendering, or fixture tests.

Restage only affected files.

Do not commit.

## Completion Report Checklist

Report:

- files changed;
- canonical active-generation status rule;
- lifecycle behavior by policy;
- diagnostic codes added or reused;
- HTML-node rejection strategy;
- escaped/code-block HTML behavior;
- metadata column-count rule;
- tests and fixtures added;
- focused test results;
- full test result;
- vet result;
- race result;
- coverage result;
- whether all three findings are resolved;
- remaining risks.

