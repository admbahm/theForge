# The Forge: Roadmap

This roadmap outlines the evolution of **The Forge** from a local-first watcher into a highly optimized, intelligent career assistant.

---

## Phase 1: Solidification & Local CLI Baseline (Current / Short Term)
*   [x] **Local CLI Parsing**: Ensure robust event loops and handling of transient write patterns from code editors (like VS Code or Obsidian auto-save).
*   [x] **Idempotency**: Prevent double-triggering of LLM queries for file system watch events that happen in close succession.
*   [x] **Context Preservation**: Establish and update `CONTEXT_MAP.md` and related metadata so developer agents can coordinate changes safely.
*   [x] **Unit & Integration Tests**: Expand filesystem testing models against mock vaults to verify permission preservation and atomic renaming.

---

## Phase 2: Compute Optimization & Advanced Local Engine (Medium Term)
*   [x] **VRAM Budgeting & Model Management**: Build interfaces to query Ollama model availability and handle memory swapping gracefully.
*   [x] **Context Truncation**: Automatically summarize very long job descriptions to fit the target model's context window bounds without discarding key requirements.
*   [x] **Concurrent Batching Pipeline**: Implement worker pools in Go to concurrently process multiple vetted jobs when first starting or executing bulk moves.
*   [x] **Local Timeout & Circuit Breaker**: Enhance external HTTP call resilience with circuit breakers to prevent stalling execution threads if the local LLM server is busy.
*   [x] **Automated Multi-Tier Funnel Orchestration**: Automate the transition between local baseline extraction (Tier 2) and premium model synthesis (Tier 3).
    *   [x] **CLI Interface**: Support running the pipeline with targeted model profiles using flags (e.g., `theforge run --tier [local|frontier|auto]`).
    *   [x] **Availability Fallback**: Programmatically test the local Ollama socket before running. If unavailable, degrade gracefully to cached summaries or report connection issues before failing.
    *   [x] **Token Compression**: Pre-process local markdown files to compress redundant text structures prior to outbound frontier API hit-testing.
    *   [x] **State Automation**: Parse frontmatter fields (`state` values: `new`, `processed`, `favorite`, `intel-ready`) or custom Obsidian tags to navigate jobs between funnel stages.

---
## Phase 3: Candidate Evidence & Application Artifact Pipeline (Next)
*   [ ] **Company Directory Organization**: Move parsed postings into dedicated company/job folders to organize clean application packets.
*   [ ] **Structured Intelligence Payloads**: Emit JSON / JSON-LD alongside markdown so downstream tools can easily query job requirements, risks, skills, and match evidence.
*   [ ] **Candidate Evidence Graph**: Implement an internal evidence mapping layer mapping job requirements to verified candidate evidence (e.g. master resume, portfolio, GitHub).
    *   **Traceability**: Downstream generators must consume this evidence layer instead of job postings directly, ensuring every resume claim or bullet point maps back to verifiable facts with no fabrication.
    *   **Confidence Scores**: Assign match confidence (direct, transferable, or gap) to requirements.
*   [ ] **Resume Tailoring Engine**: Generate custom resume variants consuming the authoritative Evidence Graph, tracing generated bullets back to verified facts.
*   [ ] **Cover Letter Generation**: Draft role-specific cover letters aligned with the verified evidence mappings and company context.
*   [ ] **Outreach & Networking Drafts**: Generate targeted recruiter outreach, hiring manager messages, and peer networking drafts.
*   [ ] **Interview Preparation Package**: Generate custom interview guides with anticipated questions, STAR-method talking points mapped to evidence, and core company themes.
*   [ ] **Unified Application Packet Exporter**: Consolidate all pipeline outputs (intelligence markdown, structured payload, tailored resume, cover letter, recruiter message, interview prep, and company research) into a single unified directory package.

---
## Phase 4: Obsidian Canvas & Downstream Analytics (Post-Pipeline)
*   [ ] **Obsidian Canvas & Graph Visualizations**: Dynamically draw relationship connections between job requirements, companies, and candidate evidence files.
*   [ ] **Dataview Dashboards**: Enable Obsidian querying and dataview analytics over structured payloads.
*   [ ] **Recurring Skill & Gap Analysis**: Aggregate and analyze recurring target skills and common requirements across target roles.
*   [ ] **Market Trend & Trend Analysis**: Analyze salary bands, hiring patterns, tech stacks, and company intelligence aggregation from the historical application vault.

---
## Phase 5: OrynCore Integration & Skill Gap Closing (Future Vision)
*   [ ] **Skill Roadmap Bridges**: Scan emerging requirements from job intelligence outputs and cross-reference them against local study roadmaps or candidate evidence.
*   [ ] **OrynCore Synchronization**: Automatically update learning backlogs and mock interview profiles to bridge candidate training gaps identified during Phase 2 parsing.
*   [ ] **Ethical Verification Loops**: Integrated mechanisms that query the candidate for proof when a new credential or skill is flagged as missing before drafting application artifacts.
