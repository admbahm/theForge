---
name: theforge-dev
description: Develop, test, debug, and review The Forge local-first Go orchestration pipeline. Use when changing its Markdown/YAML frontmatter parsing, Obsidian vault file processing, fsnotify watcher, job state transitions, Ollama integration, application payload generation, CLI, or related tests and documentation.
---

# The Forge Development

Develop The Forge without risking user vault data or documenting planned behavior as complete.

## Establish the Current Scope

- Read `README.md`, `DESIGN.md`, and the relevant source files before changing behavior.
- Inspect the current branch and working tree. Preserve unrelated edits.
- Verify whether a documented capability exists before building on it. The repository may contain mocked or incomplete pipeline phases.
- Treat OpenHunt-produced Markdown as an external data contract.

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
