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
*   [ ] **Context Truncation**: Automatically summarize very long job descriptions to fit the target model's context window bounds without discarding key requirements.
*   [ ] **Concurrent Batching Pipeline**: Implement worker pools in Go to concurrently process multiple vetted jobs when first starting or executing bulk moves.
*   [ ] **Local Timeout & Circuit Breaker**: Enhance external HTTP call resilience with circuit breakers to prevent stalling execution threads if the local LLM server is busy.
*   [ ] **Automated Multi-Tier Funnel Orchestration**: Automate the transition between local baseline extraction (Tier 2) and premium model synthesis (Tier 3).
    *   [x] **CLI Interface**: Support running the pipeline with targeted model profiles using flags (e.g., `theforge run --tier [local|frontier|auto]`).
    *   [ ] **Availability Fallback**: Programmatically test the local Ollama socket before running. If unavailable, degrade gracefully to cached summaries or report connection issues before failing.
    *   [ ] **Token Compression**: Pre-process local markdown files to compress redundant text structures prior to outbound frontier API hit-testing.
    *   [ ] **State Automation**: Parse frontmatter fields (`state` values: `new`, `processed`, `favorite`, `intel-ready`) or custom Obsidian tags to navigate jobs between funnel stages.

---

## Phase 3: Advanced Data Outputs & Obsidian Canvas (Long Term)
*   [ ] **Automatic Sub-directories**: Organically move parsed postings into dedicated sub-directories organized by company name (e.g. `Market-Insights/@Active/Netflix/`).
*   [ ] **Structured Intelligence Payloads**: Output intelligence profiles in structured formats (like JSON or JSON-LD schema) alongside the markdown notes to enable querying via Obsidian dataview or scripts.
*   [ ] **Obsidian Canvas & Graph Visualizations**: Dynamically draw relationship connections between job requirements and verified candidate evidence files in an Obsidian Canvas format.

---

## Phase 4: OrynCore Integration & Skill Gap Closing (Future Vision)
*   [ ] **Skill Roadmap Bridges**: Scan emerging requirements from job intelligence outputs and cross-reference them against local study roadmaps or candidate evidence.
*   [ ] **OrynCore Synchronization**: Automatically update learning backlogs and mock interview profiles to bridge candidate training gaps identified during Phase 2 parsing.
*   [ ] **Ethical Verification loops**: Integrated mechanisms that query the candidate for proof when a new credential or skill is flagged as missing before drafting application artifacts.
