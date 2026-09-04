---
name: jsdlc-orchestrate
description: Coordinate risk-aware software planning, implementation, QA, code review, release readiness, and incident work. Use when a user asks to plan, build, implement, fix, test, review, harden, document, prepare a release, verify a deployment, or investigate a software incident. For small factual questions or read-only explanations with no delivery work, do not invoke.
---

# Jerry SDLC orchestrator

Select the smallest workflow that provides credible evidence for the requested outcome. The user should not need to name roles or initialize repository files.

## Start

1. Read repository instructions and inspect relevant project structure without changing it.
2. Run `../../scripts/jsdlc doctor` relative to this skill directory. The wrapper verifies the bundled binary. `doctor --probe-independent` is a diagnostic only: it may observe two ephemeral Codex sessions, but must not upgrade assurance until actual role-worker receipts are bound to the run and candidate. Report the returned assurance outcome and never upgrade it based on judgment alone.
3. Classify the request and changed surface. Prefer the higher risk between deterministic triggers and reasoned judgment.
4. Briefly tell the user the workflow, risk, assurance mode, and roles being used.

If another broad orchestration skill is active, do not start a competing workflow silently. Follow an explicit user choice; otherwise report the collision.

## Assurance

- `MANAGED_INDEPENDENT`: actual role-worker receipts prove distinct execution and enforced read-only review. Phase 1 does not issue this mode yet.
- `MANAGED_SEPARATE_PASSES`: use isolated named passes, label them `SELF_REVIEW`, and never claim an independent team.
- `ADVISORY_ONLY`: provide guidance only; do not issue a `READY` verdict.
- `UNAVAILABLE`: stop the workflow and provide diagnostics.

Read [references/control-contract.md](references/control-contract.md) before any multi-role run. For release work, also read [references/release-readiness.md](references/release-readiness.md).

## Execution rules

- Roles are narrow contracts, not personas. Give each worker immutable inputs and one lens.
- Review workers are read-only. They return exact evidence and do not fix findings.
- The orchestrator adjudicates findings. A corrector receives only accepted finding IDs and bounded targets.
- Re-run affected checks after correction. A fresh verifier checks the exact final candidate.
- Scripts establish deterministic facts; agents supply judgment.
- Tool failure, stale evidence, and applicable but untestable risk produce `INCONCLUSIVE` or `BLOCKED`, never a pass.
- Never deploy, publish, tag, release, spend money, access secrets, or perform destructive operations without authority supplied by the user and allowed by repository policy.

## Completion

Report the exact candidate, checks executed, findings and dispositions, assurance mode, limitations, and one terminal status. In Phase 1, never report `READY`: even a completed review ends `INCONCLUSIVE` because evidence-backed approval is not implemented. Only a future successful `jsdlc verify` contract may authorize `READY`. The currently valid terminal statuses are `NOT_READY`, `INCONCLUSIVE`, `CANCELLED`, and `BLOCKED`.
