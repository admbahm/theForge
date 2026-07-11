# Biography Renderer Specification

The Biography renderer compiles a narrative professional profile according to length and pronoun options.

---

## 1. Length Constraints
The narrative selects different subsets of summary and experience claims based on `BiographyLength` options:
*   `short`: Summary and most recent role. Target length: 50-75 words.
*   `medium`: Summary, up to 2 roles, and 2 achievements. Target length: 100-150 words.
*   `long`: Complete summary list, up to 3 roles, multiple achievements, and education. Target length: 200-300 words.

---

## 2. Pronoun Adaptations
Leading verbs are detected (e.g. `"Served"`, `"Led"`, `"Utilized"`) and prepended with appropriate subject pronouns based on `PronounStyle`:
*   `he/him`: Prepends `"He"` (e.g. `"He served as..."`).
*   `she/her`: Prepends `"She"` (e.g. `"She served as..."`).
*   `they/them`: Prepends `"They"` (e.g. `"They served as..."`).
*   `neutral`: Prepends the candidate's name or `"The candidate"` (e.g. `"Tony Stark served as..."`).
