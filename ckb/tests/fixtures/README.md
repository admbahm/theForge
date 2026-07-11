# CKB Parser Golden Fixtures

This directory contains test input files used to verify that a Career Knowledge Base (CKB) parser and validator complies with the CKB Schema Version 1.0.

---

## 1. Valid Fixtures (`valid/`)

These files represent correctly structured, schema-compliant CKB nodes. A conforming parser must load them without error.

*   `minimal_valid.md`: Smallest valid entity (minimal metadata block, minimal headers).
*   `complete_experience.md`: Experience node showing all required headings and fields.
*   `complete_project.md`: Project node outlining objectives, design decisions, and metrics.
*   `evidence_graph.md`: File declaring an evidence registry node with relational entries.
*   `private_evidence.md`: Fictional representation of a private/redacted evidence link.
*   `many_to_many.md`: Node establishing multiple links in relation lists.

---

## 2. Invalid Fixtures (`invalid/`)

These files contain exactly one intentional syntactic or semantic defect. A conforming validator must reject these files, returning the matching error code from the error taxonomy.

*   `missing_id.md`: Metadata block is missing the `**ID**` field.
*   `malformed_id.md`: ID contains spaces or capitals.
*   `duplicate_id.md`: ID matches another node. (Handled by global validator test check).
*   `unknown_type.md`: `Type` field value is not one of the 10 allowed types.
*   `missing_required_field.md`: Missing one of the 10 required metadata fields.
*   `invalid_field_order.md`: Keys in metadata are reordered.
*   `duplicate_metadata_row.md`: Row key is declared twice.
*   `unexpected_metadata_field.md`: Contains a row key not in canonical list.
*   `invalid_enum.md`: Value not in enum lists.
*   `malformed_date.md`: Date is not YYYY-MM-DD.
*   `malformed_float.md`: Float confidence is out of bounds or not parseable.
*   `broken_relationship.md`: Relations field links to non-existent ID.
*   `invalid_relationship_type.md`: E.g. `Related Projects` links to `ev:`.
*   `duplicate_edge.md`: Listing same ID twice in relationship field.
*   `self_reference.md`: Links to its own ID.
*   `unsupported_schema_version.md`: Schema version is not `1.0`.
*   `prohibited_pii.md`: Contains an email address pattern.
