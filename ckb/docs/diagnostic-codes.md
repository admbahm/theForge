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

### 4. `CKB-METADATA-INVALID-OBJECT-TYPE`
*   **Severity**: `Fatal`
*   **Category**: Metadata
*   **Meaning**: The `Type` field is not one of the 10 allowed types.
*   **Common Cause**: spelling or casing error in type (e.g., `JobExperience` instead of `Experience`).
*   **Remediation**: Change type to one of: `Profile`, `Timeline`, `Experience`, `Project`, `Skill`, `Accomplishment`, `Credential`, `Contribution`, `Reference`, `Evidence`.
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

### 20. `CKB-STRUCTURE-MALFORMED`
*   **Severity**: `Fatal`
*   **Category**: Structure
*   **Meaning**: General markdown syntax structure error or parsing cancellation.
*   **Remediation**: Check markdown syntax or timeout configuration.
*   **Continue Parsing**: No.
