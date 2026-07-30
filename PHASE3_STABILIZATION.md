# Phase 3 Stabilization Plan

## Purpose

Phase 3 established an end-to-end integrated alpha: a job note entering `state: apply` can be matched against a validated Career Knowledge Base, compiled into a Markdown resume and cover letter, written to an application directory, and advanced to `state: completed`.

This plan defines the work required before that path is safe for routine use with a private, real-world CKB. Stabilization takes priority over new artifact types, semantic/LLM refinement, LaTeX/PDF export, and downstream analytics.

## Release Standard

Phase 3 is stable when the pipeline fails closed on unsafe configuration or evidence diagnostics, never publishes a partial packet, remains correct across duplicate events and restarts, has a hermetic verification suite, and completes a controlled end-to-end trial without exposing or inventing candidate data.

## Workstream 1: Fail-Closed Identity and CKB Configuration

**Status: Complete on `codex/phase3-stabilization`.**

### Problem

The application pipeline currently defaults to `./ckb`, which contains fictional public examples, and substitutes `Tony Stark` when no contact name is supplied. This is convenient for demos but unsafe for real applications.

### Work

- Require an explicit CKB directory for `state: apply` processing.
- Reject the committed example CKB unless an explicit demo/test mode is enabled.
- Require the minimum contact identity needed by each public artifact; never substitute a fictional person silently.
- Validate and resolve the CKB path during startup rather than during each file event.
- Add a preflight command or startup report that shows the selected vault, CKB, provider, output root, and whether demo mode is active without printing private content.
- Document a safe private-CKB setup and migration path.

### Acceptance Criteria

- Missing or example identity data cannot generate a public artifact.
- Missing, invalid, or example CKB configuration leaves the job in `state: apply` and reports an actionable error.
- Demo behavior is explicit, visually obvious, and impossible to confuse with production output.
- Tests cover missing identity, missing CKB, example CKB rejection, explicit demo mode, and valid private configuration.

## Workstream 2: Diagnostic and Evidence Enforcement

**Status: Complete on `codex/phase3-stabilization`.**

### Problem

Parser errors block generation, but planning and rendering diagnostics are not uniformly treated as release gates when a non-nil plan or artifact is returned.

### Work

- Centralize severity evaluation for parser, planner, and renderer results.
- Reject every error or fatal diagnostic before output publication.
- Define which warnings permit generation and surface them in a packet manifest.
- Ensure unsupported requirements remain gaps or transferable evidence and cannot become direct claims.
- Require complete provenance for every candidate-facing claim.
- Add negative tests for missing metrics, unsupported technologies, private evidence, strength upgrades, incomplete provenance, and conflicting claims.

### Acceptance Criteria

- No artifact is published when any stage returns a blocking diagnostic.
- Every rendered claim maps to authorized source claim IDs and source locations.
- Private/restricted evidence cannot appear in artifact bytes, logs, manifests, or diagnostics intended for public output.
- AWS-with-only-GCP/Kubernetes/Terraform and missing-metric cases pass explicit regression tests.

## Workstream 3: Transactional Application Packet Publication

**Status: Complete on `codex/phase3-stabilization`.**

### Problem

The source note is updated atomically, but artifact files are created directly. A crash can truncate an existing file or leave a partially updated packet.

### Work

- Render and export all packet files into a same-filesystem staging directory.
- Sync and close every staged file before publication.
- Include a deterministic manifest containing artifact types, digests, provenance references, warnings, source job identity, and compiler/schema versions.
- Publish the complete directory as one logical operation, preserving the previous valid packet until replacement succeeds.
- Advance the job note only after the packet is durably published.
- Record a recoverable failure without falsely advancing state.

### Acceptance Criteria

- Injected failure at any export or sync point leaves no partial current packet and does not advance the job state.
- Replacing a packet either preserves the previous complete version or publishes the new complete version.
- The manifest digests match every published artifact.
- File and directory permissions are intentional and tested.

## Workstream 4: Explicit State Machine, Idempotency, and Recovery

### Problem

State handling is distributed through conditional logic, and duplicate events are coalesced only while a path is pending or in flight. Stabilization must cover retries, restarts, and already-published packets.

### Work

- Define allowed transitions explicitly, including `apply` to `completed` and failure/retry semantics.
- Give each application build a deterministic identity derived from job input, CKB snapshot, policy, and compiler version.
- Detect an already-published identical packet and complete idempotently without rewriting it.
- Define behavior when inputs change after a packet is published.
- Test duplicate fsnotify events, editor rename patterns, partial source writes, restart during processing, and restart after publication but before state advancement.

### Acceptance Criteria

- Repeated processing of identical inputs produces one logical packet with identical bytes.
- A restart at every modeled failure boundary converges to a valid packet and correct state.
- Invalid or backward transitions are rejected and logged without modifying the note.
- Application output files do not recursively enter the job-processing pipeline.
- Packet directory names are portable, shell-safe, deterministic, and collision-resistant even when source titles contain punctuation or Unicode.

## Workstream 5: Hermetic Verification and Operational Observability

**Status: Hermetic test/race/vet gates complete; structured failure diagnostics and dry-run behavior remain in progress.**

### Problem

The complete test suite currently contains a provider-construction test that performs a real Gemini request. Operational failures are primarily log messages and are difficult to summarize or act on.

### Work

- Replace all real HTTP calls in unit and integration tests with injected transports or deterministic fakes.
- Run `go test ./...`, `go test -race ./...`, and `go vet ./...` without Ollama, network access, API keys, or a real vault.
- Add focused integration tests for every failure boundary in this plan.
- Emit structured, actionable diagnostics with job path, stage, diagnostic code, and retry guidance while avoiding private content.
- Add a dry-run/preflight mode that validates and plans without publishing artifacts or changing job state.

### Acceptance Criteria

- The full validation suite passes in a network-disabled environment.
- Race detection passes for the watcher, queue, state update, parser, planner, and renderer paths.
- Logs never include secrets or full private evidence bodies.
- A failed job clearly reports whether the user should correct configuration, CKB data, job data, or retry a transient operation.

## Workstream 6: Documentation and Controlled Alpha Trial

**Status: Operational multi-job trial complete; manual artifact, provenance, privacy, and permissions review remains open.**

### Work

- Keep `README.md`, `DESIGN.md`, `ARCHITECTURE.md`, `ROADMAP.md`, `CONTEXT_MAP.md`, `.agents/AGENTS.md`, and `HANDOFF.md` aligned with implemented behavior.
- Document backup, dry-run, packet review, retry, and rollback procedures.
- Run the complete workflow against a disposable vault and a private representative CKB.
- Manually review the generated resume, cover letter, manifest, provenance, gaps, permissions, and source-note preservation.
- Record trial findings without committing private candidate or job data.
- Normalize packet directory components so punctuation, Markdown-significant characters, Unicode dashes, traversal sequences, and collisions cannot produce awkward or unsafe output paths.

### Acceptance Criteria

- No documentation describes alpha behavior as production-ready or implemented behavior as merely planned.
- A new user can configure a private CKB and run a dry run without relying on repository examples.
- The controlled trial completes with no invented claim, private-evidence leak, partial packet, or incorrect state transition.
- Generated packet paths are portable across supported filesystems and safe to copy into shells or Markdown without escaping surprises.

## Recommended Sequence

1. Make tests hermetic so every following change has a trustworthy gate.
2. Implement fail-closed configuration and identity validation.
3. Centralize and enforce all stage diagnostics.
4. Implement transactional packet publication and manifest generation.
5. Formalize state transitions, build identity, retry, and restart recovery.
6. Complete the controlled alpha trial and close documentation gaps.

## Exit Gate

Phase 3 stabilization is complete only when all workstream acceptance criteria are met and the following commands pass without network access or external services:

```sh
go test ./...
go test -race ./...
go vet ./...
```

After this gate, the next product work should focus on deep semantic requirement mapping and structured match/gap output. LLM-assisted refinement and additional export formats should follow only after deterministic evidence enforcement remains intact through those extensions.
