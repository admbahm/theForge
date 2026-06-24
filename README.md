```text
  _______ _    _ ______   ______ ____  _____   _____ ______ 
 |__   __| |  | |  ____| |  ____/ __ \|  __ \ / ____|  ____|
    | |  | |__| | |__    | |__ | |  | | |__) | |  __| |__   
    | |  |  __  |  __|   |  __|| |  | |  _  /| | |_ |  __|  
    | |  | |  | | |____  | |   | |__| | | \ \| |__| | |____ 
    |_|  |_|  |_|______| |_|    \____/|_|  \_\\_____|______|
                                                            
```

# The Forge

**The Forge** is a local-first, event-driven career intelligence system written in Go. It supports ethical, evidence-based AI-assisted job application workflows by using the local filesystem--specifically an Obsidian vault--as its primary state-driven database.

## Project Overview

The Forge monitors your Obsidian vault for job postings ingested by "OpenHunt". It tracks the lifecycle of each selected application through frontmatter metadata, triggering local AI processing via **Ollama** (running the **Gemma 4** model) to generate job intelligence and, in planned phases, traceable evidence maps and targeted application artifacts.

The Forge is not a generic resume generator and is not intended for application spam. Its product direction is quality over volume: help candidates apply to fewer roles with stronger precision, stronger verified evidence, and better preparation.

## Core Principle

Invention of candidate experience is strictly forbidden.

Agents may reframe, emphasize, summarize, and tailor verified evidence. They must never fabricate employers, roles, dates, metrics, technologies, certifications, education, clearance status, citizenship, accomplishments, production experience, or any other candidate fact. If a job requirement is not supported by the candidate knowledge base, The Forge must identify it as a gap or transferable skill rather than claim direct experience.

## Source of Truth

Application materials should be generated only from verified source material:

- Master resume
- Achievement inventory
- Project portfolio
- Certifications
- GitHub/project evidence
- Writing samples
- Candidate preferences
- Job description

## Evidence Rules

- Every generated resume bullet must map back to one or more source facts.
- Metrics may only be used when explicitly present in source material.
- Approximate or inferred claims must be labeled internally and must not appear as hard facts.
- Transferable experience is allowed only when clearly framed as transferable.
- Concrete evidence is preferred over keyword stuffing.

For example, if a job description asks for AWS but the verified candidate knowledge base only shows GCP, Kubernetes, and Terraform experience, The Forge must not claim AWS production experience. It should frame cloud infrastructure skills as transferable and mark AWS as a gap until verified. If source material lacks a metric, The Forge must not invent percentages, dollar amounts, team sizes, uptime, or incident reduction numbers.

## 3-Phase Architecture

The following diagram illustrates the data flow and human gates within The Forge:

```mermaid
graph TD
    A[OpenHunt Crawling] -->|Inbound MD| B(Inbox Folder: #new)
    B --> C{Human Gate 1: Vetting}
    C -->|Update state to #favorite| D[The Forge Watcher]
    D -->|Trigger Ollama: Intel Generation| E[Job Intelligence Generated]
    E -->|Rewrite to Vault| F(Vault: #intel-ready)
    F --> G{Human Gate 2: Reviewing Intel}
    G -->|Update state to #apply| H[The Forge Anvil]
    H -->|Planned: Evidence Mapping| I[Evidence Map and Gap Analysis]
    I -->|Planned: Verified Artifact Drafting| J[Resume/Cover Letter/Outreach/Interview Prep]
    J --> K(Completed: #completed)
```

The currently implemented processor performs the `favorite` to `intel-ready` intelligence step. Evidence maps, tailored resumes, cover letters, recruiter messages, interview prep guides, requirement match/gap analysis, and candidate follow-up questions are part of the product model and planned artifact workflow.

## Project Structure

```text
.
├── DESIGN.md              # High-level architecture and design philosophy
├── LICENSE                # MIT License
├── README.md              # Project documentation
├── cmd
│   └── theforge
│       └── main.go        # CLI startup and signal handling
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── internal
│   └── config
│       └── config.go      # Environment and .env configuration
└── pkg
    ├── engine
    │   └── orchestrator.go # Vault watching and orchestration logic
    └── models
        └── job_post.go     # Core JobPost struct and Markdown/YAML parsing
```

## How State Management Works

The Forge implements a "Filesystem-as-Database" pattern. Instead of an external database like PostgreSQL or MongoDB, the system relies on the YAML frontmatter inside your Markdown files:

1.  **State Detection**: The `Orchestrator` uses `fsnotify` to watch for file changes in the Obsidian vault.
2.  **Schema Enforcement**: When a file is modified, The Forge unmarshals the YAML frontmatter into a `JobPost` struct.
3.  **Reactive Transitions**: If the `state` field or `favorite` boolean matches certain criteria (e.g., `state: "favorite"`), the engine triggers the corresponding Phase.
4.  **Persistence**: After processing, The Forge updates the struct's `state` (e.g., to `intel-ready`) and marshals it back into the Markdown file, preserving your content while updating the metadata.

This ensures that Obsidian remains the source of truth and the primary UI for the pipeline.

## Running The Forge

Create `.env` from the committed example and set it to the directory where OpenHunt writes Markdown files:

```sh
cp .env.example .env
go run ./cmd/theforge
```

Before The Forge will process an incoming OpenHunt note, mark it as selected by adding this field inside its YAML frontmatter:

```yaml
state: favorite
```

For example:

```yaml
---
job_id: JR333947
company: Salesforce
title: DevOps Engineer, GovCloud Mid/Senior
state: favorite
---
```

Notes without `state: favorite` are intentionally ignored. This is the human approval gate that prevents every scraped posting from being sent to Ollama. After successful intelligence generation, The Forge replaces the value with `state: intel-ready`.

The CLI loads `.env` from the current working directory unless `OPENHUNT_OUTPUT_DIR` is already exported, validates that the configured path is a directory, performs the initial scan, and watches until interrupted with `Ctrl-C` or a termination signal.

The current processor sends matching `favorite` jobs to the configured Ollama server using `gemma4:e4b`, appends a `The Forge Intelligence` section, and changes their state to `intel-ready`. The active prompt asks for role signals, evidence needs, transferable positioning, gaps, unsupported claims, candidate follow-up questions, and interview themes. Application artifact generation remains planned functionality and must use verified evidence maps before producing candidate-facing materials.

## Authors & Licensing

- **Author:** Adam Deane
- **License:** MIT License (see [LICENSE](LICENSE) for details)
