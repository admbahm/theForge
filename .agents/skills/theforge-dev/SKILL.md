---
name: theforge-dev
description: Develop, test, debug, and review The Forge local-first Go career intelligence pipeline. Use when changing its Markdown/YAML frontmatter parsing, Obsidian vault file processing, fsnotify watcher, job state transitions, Ollama integration, evidence-aware artifact generation, CLI, or related tests and documentation.
---

# The Forge Development

Develop The Forge without risking user vault data or documenting planned behavior as complete.

## Establish the Current Scope

- Read `README.md`, `DESIGN.md`, and the relevant source files before changing behavior.
- Inspect the current branch and working tree. Preserve unrelated edits.
- Verify whether a documented capability exists before building on it. The repository may contain mocked or incomplete pipeline phases.
- Treat OpenHunt-produced Markdown as an external data contract.
- Treat The Forge as a career intelligence system for ethical, evidence-based AI-assisted job applications, not a generic resume generator.

## Preserve Evidence Discipline

- Invention of candidate experience is strictly forbidden.
- Agents may reframe, emphasize, summarize, and tailor verified evidence, but must never fabricate employers, roles, dates, metrics, technologies, certifications, education, clearance status, citizenship, accomplishments, production experience, or direct experience with unsupported requirements.
- Use master resume, achievement inventory, project portfolio, certifications, GitHub/project evidence, writing samples, candidate preferences, and the job description as the planned source-of-truth set for application artifacts.
- Every generated resume bullet must map back to one or more source facts.
- Metrics may only be used when explicitly present in source material. Never invent percentages, dollar amounts, team sizes, uptime, incident reductions, or similar numbers.
- If a job asks for AWS but verified evidence only shows GCP, Kubernetes, or Terraform, do not claim AWS production experience. Frame cloud infrastructure skills as transferable and mark AWS as a gap until verified.
- Prefer concrete evidence and traceability over keyword stuffing.
- Preserve candidate authenticity, voice, constraints, and career direction.

## Implement Filesystem Changes Safely

- Preserve unknown frontmatter keys, Markdown body bytes, and existing permissions unless the requested change explicitly migrates them.
- Write updates through a same-directory temporary file followed by an atomic rename.
- Design event processing for duplicate notifications, transient partial writes, editor rename patterns, and process restarts.
- Use temporary test vaults. Never point tests at a user's real vault.

## Implement State Transitions

- Represent states and allowed transitions explicitly.
- Require the current state to match the transition source.
- Make handlers idempotent and prevent backward transitions.
- Persist generated output and the successful state transition as one logical operation.
- Record failures without falsely advancing state.

## Add External Integrations

- Put Ollama or other model access behind an interface so orchestration tests use deterministic fakes.
- Apply timeouts and cancellation to external calls.
- Keep prompts and generated sections distinguishable from source job content.
- Prompt outputs must flag unsupported claims and request missing candidate evidence instead of inventing application content.
- Do not require Ollama for parser, state-machine, or filesystem tests.

## Validate Changes

Run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

If the default Go cache is not writable, set `GOCACHE` to a directory under `/tmp`.

Add focused tests for frontmatter round trips, malformed input, state acceptance and rejection, duplicate events, atomic-write failures, permissions, and recursive watcher behavior whenever those paths change.

When prompts or artifact-generation rules change, include cases for unsupported direct requirements and missing metrics. For example, a job requiring AWS with only GCP/Kubernetes/Terraform evidence must be framed as transferable cloud infrastructure experience, and missing metrics must not become invented percentages, dollar amounts, team sizes, uptime, or incident-reduction claims.
