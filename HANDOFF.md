# Forge Handoff Document

## 1. Branch and Git Status

- **Active Branch**: `codex/master-resume-importer`
- **Base**: merged `main` at `2b876b9`
- **Checkpoint Scope**: Master-resume import, planner ordering, watcher/output isolation, OpenHunt salary compatibility, regression tests, real-vault trial findings, and aligned documentation.

## 2. Active Goal and Objectives

Close the missing boundary between a conventional Markdown master resume and The Forge's validated Career Knowledge Base so real resume and cover-letter compilation can begin under a human review gate.

## 3. Work Completed

- Added `theforge ckb import-resume -source ... -output ...`.
- Import is deterministic, local-only, source-read-only, atomic, and refuses to overwrite an existing output directory.
- Default output records are `Draft`; `-ready` explicitly creates `Active`, `Self-Attested`, 0.85-confidence records after review.
- Removes email and phone values and reports that contact fields must be configured separately.
- Generates stable experience/project IDs plus skills, education, credentials, profile, and `import-report.json` records.
- Normalizes English month/year role durations to the CKB `YYYY-MM` contract.
- Validates the complete staged CKB with strict privacy checks before publication.
- Fixed resume planning so accomplishments are associated with the selected role set independent of candidate sort order.
- Excluded `<vault>/applications` from initial scans, recursive watches, and queueing after end-to-end testing observed staged artifact events.
- Fixed watcher queue coalescing so a save received while a note is queued or processing schedules one follow-up pass instead of being lost.
- Made salary parsing compatible with OpenHunt's numeric or `unspecified` values and surfaced Markdown parse failures in watcher logs.
- Added table-driven salary contract tests plus an end-to-end orchestrator regression proving an OpenHunt `apply` note with textual missing salaries publishes both application artifacts and reaches `completed`.
- Added importer, planner integration, privacy, no-overwrite, review/ready, accomplishment-selection, date, and output-recursion tests.
- Documented the importer workflow and updated roadmap/design/agent scope.

## 4. Real-Source Verification

- Imported the real master-resume structure read-only into `/tmp`: 5 experiences and 3 projects.
- Both Draft and `-ready` imports passed strict CKB validation.
- Planned the Apple Camera Tuning job against the ready temporary CKB.
- The final plan selected 3 dated roles, 12 source accomplishments, and 10 source skills while retaining target and missing-metric warnings.
- A temporary end-to-end watcher run published `resume.md`, `cover_letter.md`, and `manifest.json`, then advanced the copied job to `state: completed`.
- A subsequent live-vault run imported the reviewed private master-resume CKB and concurrently generated separate packets for five real Apple `apply` notes, advancing each successfully processed note to `completed`.
- The live run exposed awkward packet directory names when titles contained `&`, commas, Markdown-significant punctuation, and Unicode dashes; portable collision-resistant sanitization remains open.
- Disposable verification did not modify Downloads or the Obsidian vault; the later live run was user-initiated and intentionally wrote application packets and state transitions.

## 5. Immediate Next Steps

1. Implement portable, collision-resistant packet directory naming with migration/compatibility tests.
2. Manually review live-trial claims, provenance, warnings, privacy, permissions, and source-note preservation.
3. Continue Phase 3 deterministic build identity and restart recovery.

## 6. Pending Boundaries

- The importer is a deterministic heading-based baseline, not a DOCX/PDF or arbitrary-layout parser.
- Imported facts are self-attested; it does not infer evidence, proficiency, metrics, or missing facts.
- Deep job-requirement semantic matching remains planned.
- Current packet directory sanitization retains punctuation and Unicode that are legal on macOS but awkward in shells, Markdown, and cross-platform workflows.

## 7. Validation Status

- `go test ./...`: passes.
- `go test -race ./...`: passes.
- `go vet ./...`: passes.
- `git diff --check`: passes.
