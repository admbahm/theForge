# CKB Exported JSON Model Structure

This document illustrates the stable, deterministic JSON schema output produced by the Career Knowledge Base (CKB) exporter module.

---

## 1. Output Example

```json
{
  "schema_version": "1.0",
  "objects": [
    {
      "id": "exp:acme-lead",
      "type": "Experience",
      "source_file": "ckb/experience/acme-lead.md",
      "metadata": {
        "schema_version": "1.0",
        "id": "exp:acme-lead",
        "type": "Experience",
        "status": "Active",
        "verification_level": "Independently-Verified",
        "confidence": 0.95,
        "visibility": "Public",
        "source": "HR records",
        "last_updated": "2026-07-10T00:00:00Z",
        "lifecycle_state": "Completed",
        "related_documents": [
          "timeline:main"
        ],
        "related_projects": [
          "proj:phoenix-gateway"
        ],
        "tags": [
          "golang",
          "lead"
        ]
      },
      "sections": [
        {
          "heading": "## 1. Role Context",
          "body": "* Role: Lead Systems Engineer\n..."
        }
      ],
      "relationships": [
        {
          "source_id": "exp:acme-lead",
          "target_id": "proj:phoenix-gateway",
          "type": "Related Projects",
          "source": {
            "file_path": "ckb/experience/acme-lead.md",
            "line": 1,
            "column": 0
          }
        }
      ]
    }
  ],
  "diagnostics": [
    {
      "code": "CKB-EVIDENCE-ORPHANED",
      "severity": "Warning",
      "message": "Evidence ID \"ev:unused-cert\" is never referenced",
      "source": {
        "file_path": "ckb/evidence.md",
        "line": 15,
        "column": 0
      },
      "object_id": "ev:unused-cert"
    }
  ]
}
```

---

## 2. Determinism Guarantees

*   **Stable List Ordering**: The top-level `objects` array is sorted alphabetically by `id`.
*   **Stable Relationship Ordering**: The inner `relationships` list is sorted alphabetically by `target_id`.
*   **Normalized Metadata Lists**: Comma-separated array lists are parsed, trimmed, and sorted alphabetically inside the JSON representation.
*   **Absolute Paths Excluded**: Source file properties use relative paths to remain developer-machine independent.
