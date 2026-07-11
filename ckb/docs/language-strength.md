# Language Strength Specification

To preserve factual honesty and prevent career statement inflation, the CKB module defines an explicit language-strength ranking lattice.

---

## 1. Factual Strength Lattice

Keywords and action verbs are categorized into six ranks of progressive seniority, ownership, and scope:

| Rank | Representative Keywords | Description |
| :--- | :--- | :--- |
| **Rank 1** | `observed` | Passive observation or baseline awareness. |
| **Rank 2** | `contributed`, `supported`, `trained`, `learned`, `familiar` | Collaborative assistance or educational exposure. |
| **Rank 3** | `implemented`, `proficient`, `estimated`, `targeted`, `planned` | Direct execution, creation, or local project planning. |
| **Rank 4** | `delivered`, `measured` | Completed release delivery and verified outcome measuring. |
| **Rank 5** | `led`, `owned`, `expert` | Absolute ownership or leadership of initiative. |
| **Rank 6** | `directed` | Senior administrative governance or executive direction. |

---

## 2. Invariant Enforcement
*   A downstream adaptation or rendering step **MAY** preserve or weaken statement strength (e.g. changing `"Led"` to `"Contributed"` for brevity).
*   It **MUST NOT** upgrade strength (e.g. converting `"Contributed"` to `"Led"`).
*   Any statement containing a higher rank verb than its source claim triggers a `CKB-RENDER-STRENGTH-UPGRADE` diagnostic warning.
