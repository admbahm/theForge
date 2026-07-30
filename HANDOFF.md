# Forge Handoff Document

## 1. Branch and Git Status

- **Active Branch**: `codex/phase3-stabilization`
- **Base**: `main`
- **Working Tree**: Clean at the Workstream 2/3 checkpoint when this handoff was written.
- **Checkpoint Scope**: Evidence regressions and transactional application packet publication with deterministic manifests and rollback coverage.

## 2. Active Goal and Objectives

Complete the release gates in `PHASE3_STABILIZATION.md` so the `apply` to `completed` application pipeline can be used safely with a private, real-world Career Knowledge Base.

The stabilization sequence is:

1. Establish hermetic test, race, and vet gates.
2. Fail closed on missing/example CKB and identity configuration.
3. Enforce blocking diagnostics from every compilation stage.
4. Publish application packets transactionally with a manifest.
5. Formalize state transitions, idempotency, and restart recovery.
6. Complete a controlled disposable-vault trial and align documentation.

## 3. Work Completed

- Created `codex/phase3-stabilization` from `main`.
- Removed Ollama model-availability network I/O from `llm.NewClient`; runtime connectivity remains checked explicitly by `run` for the local and auto tiers.
- Replaced the Gemini factory test's live API request with structural assertions that verify provider type, configured model, and environment-sourced API key.
- Established passing network-independent test, race, and vet gates.
- Added YAML and environment configuration for a private application CKB, demo mode, and candidate contact identity, with environment values taking precedence.
- Resolve and validate a configured CKB during startup; reject the bundled fictional CKB unless explicit demo mode is enabled.
- Removed the implicit `./ckb` and `Tony Stark` fallbacks from application processing.
- Require candidate name and email before any public resume or cover letter can be published.
- Leave rejected jobs byte-for-byte unchanged in `state: apply` and create no application output directory.
- Add an unmistakable fictional-data warning to every demo resume and cover letter.
- Added a startup preflight report for the vault, CKB status, output root, provider, tier, and demo mode without printing candidate evidence.
- Documented safe private application configuration in the examples and README.
- Centralized blocking diagnostic severity and stable-code extraction in `ckb/model`.
- Enforced parser, resume/cover-letter planner, and resume/cover-letter renderer error/fatal diagnostics before artifact publication, even when a stage returns a non-nil plan or artifact.
- Restricted orchestration errors to stable diagnostic codes so private evidence and diagnostic message bodies are not copied into operational logs.
- Added explicit regressions proving AWS remains a gap when only GCP/Kubernetes/Terraform evidence exists and missing source metrics are not invented during planning/rendering.
- Replaced direct artifact writes with secure same-filesystem packet staging and publication.
- Sync every staged artifact, the deterministic manifest, and staging directory before publication.
- Added `manifest.json` with SHA-256 file digests, sizes, artifact content digests, provenance references, warning codes, source-job identity, schema/compiler versions, and demo status.
- Publish packets by directory rename with rollback to the previous complete packet when replacement fails.
- Use `0700` packet directories and `0600` packet files.
- Added failure-injection coverage for file writes, staging-directory sync, initial publication rename, and replacement rollback.
- Updated project documentation to reflect completed Workstreams 1–3 accurately.

## 4. Immediate Next Steps

Begin Workstream 4, explicit state machine, idempotency, and recovery:

1. Define allowed transitions and reject invalid/backward transitions without modifying notes.
2. Derive a deterministic application build identity from job input, CKB snapshot, policy, and compiler version.
3. Record the build identity in `manifest.json` and detect an identical published packet.
4. Complete `state: apply` to `completed` idempotently after a restart that occurs after packet publication but before note advancement.
5. Add restart/failure-boundary and output-recursion tests.

## 5. Pending Decisions and Blockers

- Packet replacement currently preserves the previous packet during publication and removes it after the new packet and parent directory are synced. Long-term packet history/version retention remains undecided.
- Recoverable build failures currently leave `state: apply` unchanged and log an actionable error; a persistent diagnostic record remains undecided.
- No current implementation blocker.

## 6. Validation Status

Run with `GOCACHE=/tmp/theforge-go-cache` because the default cache is not writable in the managed environment:

- `go test ./...`: passes.
- `go test -race ./...`: passes.
- `go vet ./...`: passes.
- No test requires network access, Ollama, API keys, or a real vault.
