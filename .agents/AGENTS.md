# Workspace Rules for The Forge

These rules apply to all work in this repository.

## Project Scope

- Treat The Forge as a local-first Go application that uses Markdown files with YAML frontmatter as its state store.
- Keep the Obsidian vault human-readable and usable without The Forge running.
- Treat The Forge as a career intelligence system for ethical, evidence-based AI-assisted job applications, not as a generic resume generator.
- Distinguish implemented behavior from planned behavior. Ollama integration, the multi-tier funnel CLI flags (`local`, `frontier`, `auto` tiers), and the `new` to `processed` / `favorite` to `intel-ready` transitions are implemented; evidence mapping and application artifact generation remain planned.
- Maintain compatibility with the OpenHunt frontmatter contract documented in `README.md` and `DESIGN.md`.

## Evidence and Product Rules

- Invention of candidate experience is strictly forbidden.
- Agents may reframe, emphasize, summarize, and tailor verified evidence, but must never fabricate employers, roles, dates, metrics, technologies, certifications, education, clearance status, citizenship, accomplishments, or production experience.
- Treat master resume, achievement inventory, project portfolio, certifications, GitHub/project evidence, writing samples, candidate preferences, and the job description as the source-of-truth set for future application artifacts.
- Every generated resume bullet must map back to one or more source facts.
- Metrics may only be used when explicitly present in source material; never invent percentages, dollar amounts, team sizes, uptime, incident reductions, or similar numbers.
- If a job requirement is not supported by verified evidence, identify it as a gap or transferable skill instead of claiming direct experience.
- Prefer concrete evidence and hiring-manager usefulness over keyword stuffing.
- Preserve candidate authenticity, voice, constraints, and career direction.

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
- Add or update tests when prompts or artifact generation rules change, including unsupported technology requirements and missing metric cases.

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
