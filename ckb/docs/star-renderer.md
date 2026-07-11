# STAR Story Renderer Specification

The STAR story renderer structures achievement blocks into Situation, Task, Action, and Result components.

---

## 1. Structure Requirements
*   **Situation**: Identifies the context or problem.
*   **Task**: Defines the challenge or goal.
*   **Action**: Specifies the candidate's exact actions.
*   **Result**: Highlights the quantifiable output.

---

## 2. Incompleteness Handling Policies
If any of the four components is missing in the plan for a story, the renderer respects the `IncompleteSTARBehavior` choice:
*   `Fail`: Aborts rendering with a fatal diagnostic error (`CKB-RENDER-INCOMPLETE-STAR`).
*   `Skip`: Omits the incomplete story section silently from the final artifact, adding a warning diagnostic.
*   `WarnAndRender`: Renders the available components and appends a warning warning of missing details.

---

## 3. Formatting Formats
*   `labeled`: Situation, Task, Action, Result prefixed lists.
*   `paragraph`: Unified narrative paragraph.
*   `bullet`: Concise list format.
