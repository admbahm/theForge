# Forge Handoff Document

## 1. Branch and Git Status

- **Active Branch**: `codex/phase3-stabilization`
- **Base**: `main`
- **Working Tree**: Clean at the stabilization checkpoint when this handoff was written.
- **Checkpoint Scope**: Hermetic provider tests, fail-closed application configuration, explicit demo safeguards, and centralized blocking diagnostic enforcement.

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

## 4. Immediate Next Steps

Complete the remaining diagnostic regression coverage, then begin Workstream 3:

1. Add an explicit AWS-required / GCP-Kubernetes-Terraform-only regression that proves AWS cannot become a direct claim.
2. Add a missing-source-metric regression at the complete planning/rendering boundary.
3. Carry publishable warning diagnostics into Workstream 3's packet manifest.
4. Design and implement same-filesystem staged packet publication with digests and atomic replacement semantics.

## 5. Pending Decisions and Blockers

- Packet replacement/history semantics and recoverable failure representation remain open for later workstreams.
- Warning diagnostics need a manifest destination; full warning publication depends on Workstream 3's packet manifest.
- No current implementation blocker.

## 6. Validation Status

Run with `GOCACHE=/tmp/theforge-go-cache` because the default cache is not writable in the managed environment:

- `go test ./...`: passes.
- `go test -race ./...`: passes.
- `go vet ./...`: passes.
- No test requires network access, Ollama, API keys, or a real vault.
