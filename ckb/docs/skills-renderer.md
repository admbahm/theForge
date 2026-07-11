# Skills Summary Renderer Specification

The Skills Summary renderer compiles tech and leadership categories with verified evidence backing and duration intervals.

---

## 1. Grouping and Structuring
*   Skills are classified into **Technical Skills** (technologies, languages, tools) and **Leadership Skills** (methodologies, management, systems).
*   Each skill is backed by exact evidence citations and the number of source files referencing it.

---

## 2. Non-Overlapping Duration Algorithm
*   To calculate skill duration:
    1. Gather all roles referencing the skill.
    2. Sort their start and end dates chronologically.
    3. Union overlapping time ranges to form a set of non-overlapping intervals.
    4. Sum the total months across unioned intervals.
*   This prevents double-counting overlapping dates (e.g. concurrent roles).
