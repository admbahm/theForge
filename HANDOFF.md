# Forge Handoff Document

This document records the current state of the repository, completed objectives, and next steps for subsequent agent sessions.

## 1. Branch & Git Status
* **Active Branch**: `feat/phase3-integration`
* **Working Tree**: Clean.

## 2. Active Goal & Objectives
* **Active Goal**: Implement Phase 3 (Pipeline & Artifact Integration) to automate evidence-grounded resume, CV, and cover letter generation inside the fsnotify watcher pipeline.
* **Scope**: 
  1. Extend `JobPost` state transitions to include `state: apply` and `state: completed`.
  2. Modify the `Orchestrator` watcher event loop to intercept `state: apply`.
  3. Load the CKB and run the planning and rendering logic when `state: apply` is detected.
  4. Write generated artifacts (tailored resumes, cover letters) to a dedicated application directory.
  5. Atomically advance the job post's state to `completed`.
  6. Implement the deterministic Cover Letter generation module.

## 3. Work Completed
* **Pre-Phase 3 Milestones Merged**: Pre-Phase 3 (OpenAI/Gemini HTTP clients, prompt refactorings, and `ckb` CLI validate/export/plan commands) has been successfully merged into `main` (PR #15).
* **Workspace Configuration**: Added Git Flow & Branching rules and Agent Session Handoff Discipline to [.agents/AGENTS.md](file:///Users/adam/dev/cross/TheForge/.agents/AGENTS.md).
* **Branching**: Branch `feat/phase3-integration` has been checked out as the active working branch.

## 4. Immediate Next Steps
1. **Extend JobPost Model**: Add states `apply` and `completed` to job post logic.
2. **Update fsnotify Orchestrator**: Update `pkg/engine/orchestrator.go` to support loading the CKB, running `planning.BuildPlan` and `rendering.Render` for `state: apply` postings.
3. **Design Cover Letter Renderer**: Build a Cover Letter generation compiler inside `ckb/rendering/`.
4. **Define Directory Exporter**: Output all generated application materials into `<vault_path>/applications/<company>-<role>/`.

## 5. Pending Open Questions & Blockers
* **Cover Letter Template format**: Do we want to support Markdown/LaTeX formats for cover letters, similar to resumes?
* **Application Folder Structure**: Does outputting packages to `<vault_path>/applications/<company>-<role>/` align with your Obsidian setup?
