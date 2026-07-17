# Forge Handoff Document

This document records the current repository state and the next actionable work. It reflects the post-merge state of Phase 3 and supersedes the feature-branch handoff from its implementation session.

## 1. Branch and Git Status

- **Active Branch**: `codex/phase3-stabilization-docs`
- **Base**: `main` at Phase 3 integration commit `39b2d1d`
- **Working Tree**: Documentation changes are present and uncommitted in `.agents/AGENTS.md`, `ARCHITECTURE.md`, `CONTEXT_MAP.md`, `CONTRIBUTING.md`, `DESIGN.md`, `README.md`, `ROADMAP.md`, and `HANDOFF.md`; `PHASE3_STABILIZATION.md` is new.

## 2. Active Goal and Objectives

- Reconcile project documentation with the Phase 3 implementation that has landed on `main`.
- Classify the application artifact pipeline accurately as an integrated alpha.
- Define the safety and operational work required before routine use with a private real-world CKB.

## 3. Current Implemented State

- The multi-tier watcher supports `new` to `processed` and `favorite` to `intel-ready` intelligence transitions.
- Ollama, OpenAI, and Gemini provider clients are implemented; Ollama remains the local default.
- The CKB subsystem parses and validates structured Markdown, builds a knowledge graph, extracts and authorizes claims, plans artifacts under policy, and deterministically renders/exports artifacts with provenance.
- A job entering `state: apply` invokes the CKB pipeline, writes a Markdown resume and cover letter under `<vault>/applications/<company>-<role>/`, preserves the source note, and advances it to `state: completed` after generation.
- Unit, validation, fuzz, renderer, and apply-path integration coverage exists. `go vet ./...` passes in the current environment.

## 4. Known Stabilization Findings

- Application processing defaults to the repository's fictional example CKB and falls back to the fictional name `Tony Stark`; real artifact generation must instead fail closed on missing identity/CKB configuration.
- Resume and cover-letter files are written directly rather than published as one transactional packet.
- The orchestrator checks parser diagnostics but does not uniformly reject blocking planner and renderer diagnostics when a non-nil result exists.
- State/event handling needs explicit retry, restart, and already-published-packet semantics.
- `go test ./...` is not hermetic: `TestNewClientCreatesConfiguredProvider` currently performs a real Gemini request and fails without network access or with a different remote error response.
- Deep semantic requirement-to-evidence matching, explicit direct/transferable/gap output, recruiter outreach, interview preparation, structured packet payloads, and additional export formats remain planned.

## 5. Work Completed on This Branch

- Added `PHASE3_STABILIZATION.md` with six workstreams, acceptance criteria, recommended sequencing, and an explicit exit gate.
- Updated the README, design, architecture, roadmap, context map, contributing guidance, and agent instructions to distinguish shipped Phase 3 behavior from remaining stabilization and future capabilities.
- Removed stale claims that OpenAI/Gemini or application artifact generation are merely planned.
- Documented the current alpha risks instead of presenting Phase 3 as production-ready.

## 6. Immediate Next Steps

Follow `PHASE3_STABILIZATION.md` in this order:

1. Replace the network-dependent provider test with an injected fake transport and establish hermetic `test`, race, and vet gates.
2. Require explicit private CKB and candidate identity configuration; make demo/example behavior opt-in.
3. Centralize and enforce parser, planner, and renderer blocking diagnostics.
4. Stage, sync, manifest, and transactionally publish complete application packets before advancing job state.
5. Formalize state transitions, deterministic build identity, retry behavior, and restart recovery.
6. Run a controlled trial against a disposable vault and private representative CKB without committing private data.

## 7. Pending Decisions and Blockers

- Decide whether private CKB configuration belongs in `theforge.yaml`, an environment variable, or both with a documented precedence order.
- Define the minimum required contact fields for each public artifact type.
- Choose packet replacement/history semantics when the job, CKB, policy, or compiler version changes.
- Decide whether a recoverable build failure needs a new frontmatter state or should leave `state: apply` with a separate diagnostic field/file.
- Define which warning diagnostics are publishable and how they appear in the packet manifest.

## 8. Validation Status

- Documentation-only edits do not change Go behavior.
- `go vet ./...`: passes.
- `go test ./...`: currently fails in `internal/llm` because `TestNewClientCreatesConfiguredProvider` makes a real Gemini request; this is recorded as the first stabilization item rather than hidden as an environment-only failure.
