# CKB Diagnostic Error Codes Catalog

This document defines the stable, public diagnostic error codes emitted by the Career Knowledge Base (CKB) parser and graph validator.

---

## Code Catalog

### 1. `CKB-METADATA-MISSING-ID`
*   **Severity**: `Fatal`
*   **Category**: Metadata
*   **Meaning**: The metadata block is missing the `**ID**` field, or the table is completely missing.
*   **Common Cause**: Typo in the table key or the document doesn't start with a Markdown table.
*   **Remediation**: Add a `**ID**` row in the metadata table.
*   **Continue Parsing**: No (fatal).

### 2. `CKB-IDENTITY-INVALID-ID`
*   **Severity**: `Fatal`
*   **Category**: Identity
*   **Meaning**: The object ID does not match the required `^[a-z0-9]+:[a-z0-9-]+$` format.
*   **Common Cause**: ID contains uppercase letters, underscores, spaces, or is missing the type prefix.
*   **Remediation**: Rename the ID using lowercase alphanumeric characters, colons, and hyphens (e.g. `exp:stark-devops`).
*   **Continue Parsing**: No (fatal).

### 3. `CKB-IDENTITY-DUPLICATE-ID`
*   **Severity**: `Fatal`
*   **Category**: Identity
*   **Meaning**: The declared ID has already been registered by another file.
*   **Common Cause**: Copy-pasting a file without updating the ID field.
*   **Remediation**: Ensure all CKB files declare globally unique IDs.
*   **Continue Parsing**: No (fatal).

### 3A. `CKB-IDENTITY-PREFIX-TYPE-MISMATCH`
*   **Severity**: `Fatal`
*   **Category**: Identity
*   **Meaning**: The ID prefix does not match the declared object `Type`.
*   **Common Cause**: Copying a metadata table and changing `Type` without changing `ID`, such as `ID: exp:foo` with `Type: Project`.
*   **Remediation**: Use the canonical type prefix from the schema, for example `proj:` for `Project` or `edu:` for `Education`.
*   **Continue Parsing**: No.

### 4. `CKB-METADATA-INVALID-OBJECT-TYPE`
*   **Severity**: `Fatal`
*   **Category**: Metadata
*   **Meaning**: The `Type` field is not one of the allowed object types.
*   **Common Cause**: spelling or casing error in type (e.g., `JobExperience` instead of `Experience`).
*   **Remediation**: Change type to one of: `Profile`, `Timeline`, `Experience`, `Project`, `Skill`, `Accomplishment`, `Credential`, `Contribution`, `Reference`, `Evidence`, `Education`.
*   **Continue Parsing**: No (fatal).

### 5. `CKB-METADATA-MISSING-FIELD`
*   **Severity**: `Fatal`
*   **Category**: Metadata
*   **Meaning**: The table is missing one of the 10 required metadata keys.
*   **Common Cause**: A required field row was accidentally deleted.
*   **Remediation**: Add the missing required field matching the canonical schema.
*   **Continue Parsing**: Yes (when optional fields are missing; fatal if missing required metadata block).

### 6. `CKB-METADATA-INVALID-ORDER`
*   **Severity**: `Error`
*   **Category**: Metadata
*   **Meaning**: The keys in the metadata table do not match the canonical ordering sequence.
*   **Common Cause**: Metadata table keys were reordered or inserted out of order.
*   **Remediation**: Align metadata table keys in the exact order documented in `metadata.md`.
*   **Continue Parsing**: Yes.

### 7. `CKB-METADATA-DUPLICATE-FIELD`
*   **Severity**: `Fatal`
*   **Category**: Metadata
*   **Meaning**: A metadata field key is declared twice in the table.
*   **Common Cause**: Duplicate rows.
*   **Remediation**: Remove the duplicate table row.
*   **Continue Parsing**: No.

### 8. `CKB-METADATA-UNKNOWN-FIELD`
*   **Severity**: `Error`
*   **Category**: Metadata
*   **Meaning**: The metadata table contains a key that is not in the canonical required or optional lists.
*   **Common Cause**: Spelling mistake or typo in a key name.
*   **Remediation**: Rename or remove the unexpected row field.
*   **Continue Parsing**: Yes.

### 9. `CKB-METADATA-INVALID-ENUM`
*   **Severity**: `Error`
*   **Category**: Metadata
*   **Meaning**: An enum value does not match one of the allowed categories.
*   **Common Cause**: Spelling or casing issue (e.g. `public` instead of `Public`).
*   **Remediation**: Correct enum value to match allowed cases exactly.
*   **Continue Parsing**: Yes.

### 10. `CKB-METADATA-INVALID-DATE`
*   **Severity**: `Error`
*   **Category**: Metadata
*   **Meaning**: The `Last Updated` date format does not match `YYYY-MM-DD`.
*   **Common Cause**: Using letters or slash separators (e.g. `July 10, 2026`).
*   **Remediation**: Format date exactly as `YYYY-MM-DD`.
*   **Continue Parsing**: Yes.

### 11. `CKB-METADATA-INVALID-CONFIDENCE`
*   **Severity**: `Error`
*   **Category**: Metadata
*   **Meaning**: The confidence score is not a valid float between `0.00` and `1.00`.
*   **Common Cause**: Floats out of range (e.g. `1.50`).
*   **Remediation**: Ensure confidence is a float from `0.00` to `1.00`.
*   **Continue Parsing**: Yes.

### 12. `CKB-GRAPH-BROKEN-REFERENCE`
*   **Severity**: `Error`
*   **Category**: Graph
*   **Meaning**: A relation links to an ID that does not exist in the vault.
*   **Common Cause**: Typo in the related ID field or the target file was deleted.
*   **Remediation**: Fix the target ID in the relation or add the target node.
*   **Continue Parsing**: Yes.

### 13. `CKB-RELATIONSHIP-INVALID-TYPE`
*   **Severity**: `Error`
*   **Category**: Graph
*   **Meaning**: The prefix of the target ID does not match the relationship type requirement.
*   **Common Cause**: Linking to `ev:` prefix under `Related Projects` instead of `proj:`.
*   **Remediation**: Correct relationship list or verify target ID prefix.
*   **Continue Parsing**: Yes.

### 14. `CKB-GRAPH-DUPLICATE-EDGE`
*   **Severity**: `Error`
*   **Category**: Graph
*   **Meaning**: A relation list references the same ID twice.
*   **Common Cause**: Duplicate listing in comma-separated list.
*   **Remediation**: De-duplicate the list values.
*   **Continue Parsing**: Yes.

### 15. `CKB-GRAPH-INVALID-SELF-REFERENCE`
*   **Severity**: `Error`
*   **Category**: Graph
*   **Meaning**: An object links to its own ID.
*   **Common Cause**: Listing self ID in related field.
*   **Remediation**: Remove self ID from the list.
*   **Continue Parsing**: Yes.

### 16. `CKB-EVIDENCE-ORPHANED`
*   **Severity**: `Warning`
*   **Category**: Evidence
*   **Meaning**: An evidence node is registered in catalog tables but never linked by any experience or project.
*   **Common Cause**: Declaring evidence that is not yet linked.
*   **Remediation**: Reference the evidence in your experiences/projects metadata blocks, or remove it.
*   **Continue Parsing**: Yes.

### 16A. `CKB-EVIDENCE-UNRESOLVED`
*   **Severity**: `Warning`
*   **Category**: Evidence
*   **Meaning**: A selected claim referenced evidence that could not be authorized as resolved, policy-permitted provenance.
*   **Common Cause**: Typo, deleted evidence row, malformed evidence ID, wrong target type, restricted visibility, or ineligible evidence state.
*   **Remediation**: Fix the evidence reference or add a resolvable Evidence catalog row with policy-permitted visibility before relying on it as provenance.
*   **Continue Parsing**: Yes. The unsafe evidence ID is removed from public outputs.

### 17. `CKB-VERSION-UNSUPPORTED`
*   **Severity**: `Error`
*   **Category**: Version
*   **Meaning**: The declared Schema Version is not supported.
*   **Common Cause**: Schema version declared as `2.0`.
*   **Remediation**: Revert version declaration to `1.0`.
*   **Continue Parsing**: Yes.

### 18. `CKB-PRIVACY-PROHIBITED-PII`
*   **Severity**: `Fatal`
*   **Category**: Privacy
*   **Meaning**: A file contains prohibited email or phone number pattern.
*   **Common Cause**: Forgetting to sanitize resume details.
*   **Remediation**: Remove the PII from CKB markdown files.
*   **Continue Parsing**: No (fatal).

### 19. `CKB-LIMITS-EXCEEDED`
*   **Severity**: `Fatal`
*   **Category**: Security
*   **Meaning**: A parsed document or repository exceeded configured size/counts safety limits.
*   **Common Cause**: File too large or too many relationships.
*   **Remediation**: Reduce file sizing or partition project directories.
*   **Continue Parsing**: No.

### 19A. `CKB-LIMIT-RELATIONSHIPS-EXCEEDED`
*   **Severity**: `Fatal`
*   **Category**: Security
*   **Meaning**: A single object declares more relationship targets than `MaxRelationshipsNode`.
*   **Common Cause**: Very large comma-separated relationship lists, or duplicate targets repeated in relationship metadata rows.
*   **Remediation**: Reduce declared relationship fanout or split the source record. Duplicate declared targets count toward the limit before duplicate-edge validation.
*   **Continue Parsing**: No.

### 20. `CKB-STRUCTURE-MALFORMED`
*   **Severity**: `Fatal`
*   **Category**: Structure
*   **Meaning**: General markdown syntax structure error or parsing cancellation.
*   **Remediation**: Check markdown syntax or timeout configuration.
*   **Continue Parsing**: No.

### 21. `CKB-PLAN-CONFLICT-DETECTED`
*   **Severity**: `Error`
*   **Category**: Planning
*   **Meaning**: Conflicting claims detected during target planning.
*   **Remediation**: Add an override block inside PlanRequest to resolve the conflict.

### 22. `CKB-PLAN-PROVENANCE-INCOMPLETE`
*   **Severity**: `Fatal`
*   **Category**: Planning
*   **Meaning**: Selected claim lacks required source graph links or files.
*   **Remediation**: Re-run parser to verify node graph integrity.

### 23. `CKB-RENDER-INVALID-PLAN`
*   **Severity**: `Fatal`
*   **Category**: Rendering
*   **Meaning**: Passed plan is nil or contains a malformed version format.
*   **Remediation**: Re-compile plan using the planning builder.

### 24. `CKB-RENDER-UNSUPPORTED-ARTIFACT`
*   **Severity**: `Fatal`
*   **Category**: Rendering
*   **Meaning**: Requested target artifact type is not supported.
*   **Remediation**: Set type to one of: resume, cv, biography, star-story, skills-summary.

### 25. `CKB-RENDER-MISSING-PROVENANCE`
*   **Severity**: `Fatal`
*   **Category**: Rendering
*   **Meaning**: A claim selected in the plan lacks any source file provenance.
*   **Remediation**: Correct source path tracking in planner registry.

### 26. `CKB-RENDER-VISIBILITY-VIOLATION`
*   **Severity**: `Fatal`
*   **Category**: Rendering
*   **Meaning**: Confidential claims are about to leak in public scopes.
*   **Remediation**: Ensure policy AllowedVisibilities match the claims.

### 27. `CKB-RENDER-BLOCKING-CONFLICT`
*   **Severity**: `Fatal`
*   **Category**: Rendering
*   **Meaning**: Render is halted due to unresolved active conflicts.
*   **Remediation**: Declare overrides to resolve conflict.

### 27A. `CKB-RENDER-NONBLOCKING-CONFLICT`
*   **Severity**: `Warning`
*   **Category**: Rendering
*   **Meaning**: A non-blocking planning conflict was retained for audit visibility while rendering continued.
*   **Remediation**: Review the source claims if the warning affects the target artifact.

### 28. `CKB-RENDER-INCOMPLETE-STAR`
*   **Severity**: `Fatal` / `Warning`
*   **Category**: Rendering
*   **Meaning**: A STAR story is missing situation/task/action/result.
*   **Remediation**: Add missing components in CKB experience markdown.

### 29. `CKB-RENDER-BUDGET-EXCEEDED`
*   **Severity**: `Warning`
*   **Category**: Rendering
*   **Meaning**: Configured count budgets exceeded.
*   **Remediation**: Reduce target limits or select fewer items in plan.

### 30. `CKB-RENDER-STRENGTH-UPGRADE`
*   **Severity**: `Warning`
*   **Category**: Rendering
*   **Meaning**: Adapted prose keyword ranks higher than source claim.
*   **Remediation**: Weaken or preserve original action verbs.
