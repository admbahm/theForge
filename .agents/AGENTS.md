# Workspace Rules for The Forge

These rules apply to all work in this repository.

## Project Scope

- Treat The Forge as a local-first Go application that uses Markdown files with YAML frontmatter as its state store.
- Keep the Obsidian vault human-readable and usable without The Forge running.
- Distinguish implemented behavior from planned behavior. Ollama integration, intelligence generation, and application payload generation are not implemented yet.
- Maintain compatibility with the OpenHunt frontmatter contract documented in `README.md` and `DESIGN.md`.

## Go Development

- Use the Go version declared in `go.mod`.
- Format changed Go files with `gofmt`.
- Prefer standard-library packages unless a dependency materially simplifies the implementation.
- Keep domain models independent from filesystem watching and AI-provider concerns.
- Replace deprecated `io/ioutil` usage when modifying affected code.
- Propagate actionable errors; do not silently discard malformed job files without an intentional logging policy.

## Filesystem and Frontmatter Safety

- Never overwrite a vault file in place when a failed write could truncate it. Write a temporary file in the same directory, sync it when appropriate, then rename it atomically.
- Preserve file permissions.
- Preserve Markdown body content and unknown YAML fields unless a migration explicitly changes them.
- Treat editor writes and filesystem notifications as noisy: account for duplicate events, partial writes, renames, and retries.
- Add recursive watches for directories whose files must be processed after startup.

## State Machine

- Define states and allowed transitions explicitly; do not rely on unrelated flags after a job has advanced.
- Make processing idempotent so repeated filesystem events cannot move a job backward or duplicate generated content.
- Validate required fields before performing a transition.
- Add tests for every new transition and rejection path.

## Validation

Before completing a code change, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

Use a workspace-local or `/tmp` `GOCACHE` if the environment cannot write to the default Go build cache.

For filesystem behavior, use temporary directories and verify both resulting content and permissions. Do not test against a real Obsidian vault.

## Change Discipline

- Keep documentation aligned with actual behavior and repository structure.
- Avoid committing generated job data, private vault content, local configuration, credentials, IDE state, binaries, coverage output, or temporary files.
- Do not modify unrelated user changes in the working tree.
