# Forge Handoff Document

## 1. Branch and Git Status

- **Active Branch**: `codex/application-context-headers`
- **Base**: `main` at `0c6a193`
- **Checkpoint Scope**: Application-target context in generated resume and cover-letter Markdown, source-path privacy, regression tests, and aligned documentation/instructions.

## 2. Active Goal and Objectives

Make every generated resume and cover letter immediately identifiable during review without moving or duplicating the source job note.

## 3. Work Completed

- Prepended a removable `Application Target — Internal` callout to generated resume and cover-letter Markdown.
- Included company, role, optional job ID/location, and the vault-relative source-note path in the callout.
- Prevented unexpected out-of-vault source paths from exposing absolute paths by falling back to the source basename.
- Covered context presence and source-path privacy with orchestrator regression tests.
- Aligned README, design, architecture, context map, roadmap, stabilization plan, and agent-development instructions with the behavior.
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

1. Manually verify the callout in a disposable generated packet and confirm the removal step fits the submission workflow.
2. Complete portable packet-component sanitization and migration/compatibility tests; deterministic collision-resistant suffixes are implemented.
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
