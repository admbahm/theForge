# CV Renderer Specification

The CV renderer compiles a comprehensive Curriculum Vitae (CV) from all selected claims.

---

## 1. Subtype Ordering Rules

The section order of the CV is dynamically re-sorted depending on the selected subtype:

*   `professional`: Summary -> Experience -> Skills -> Education -> Credentials -> Contributions.
*   `technical`: Summary -> Skills -> Experience -> Education -> Credentials -> Contributions.
*   `executive`: Summary -> Experience -> Skills -> Education -> Credentials -> Contributions.
*   `academic-adjacent`: Education -> Contributions -> Summary -> Experience -> Skills -> Credentials.

---

## 2. Content Sections
*   **Professional Profile (Summary)**: Objective statement or career summary.
*   **Detailed Experience**: Complete promotion history and achievement lists.
*   **Skills**: Detailed skill bullet points.
*   **Education History**: Academic degrees and institutions.
*   **Professional Certifications**: Credentials and licenses.
*   **Professional Contributions**: Publications, speaking engagements, and mentoring records.
