---
name: jsdlc-orchestrate
description: Coordinate risk-aware software planning, implementation, QA, code review, release readiness, and incident work. Use when a user asks to plan, build, implement, fix, test, review, harden, document, prepare a release, verify a deployment, or investigate a software incident. For small factual questions or read-only explanations with no delivery work, do not invoke.
---

# Jerry SDLC orchestrator

Select the smallest workflow that provides credible evidence for the requested outcome. The user should not need to name roles or initialize repository files.

## Start

1. Read repository instructions and inspect relevant project structure without changing it.
2. Run `../../scripts/jsdlc doctor` relative to this skill directory. The wrapper verifies the bundled binary. `doctor --probe-independent` is a diagnostic only and never upgrades stored assurance.
3. Classify the request and changed surface. Prefer the higher risk between deterministic triggers and reasoned judgment.
4. Briefly tell the user the workflow, risk, assurance mode, and roles being used.

If another broad orchestration skill is active, do not start a competing workflow silently. Follow an explicit user choice; otherwise report the collision.

## Assurance

- `MANAGED_INDEPENDENT`: reserved until a trusted runtime adapter can attest worker identity and isolation. Distinct IDs in CLI output are insufficient.
- `MANAGED_SEPARATE_PASSES`: use isolated named passes, label them `SELF_REVIEW`, and never claim an independent team.
- `ADVISORY_ONLY`: provide guidance only; do not issue a `READY` verdict.
- `UNAVAILABLE`: stop the workflow and provide diagnostics.

Read [references/control-contract.md](references/control-contract.md) before any multi-role run. For release work, also read [references/release-readiness.md](references/release-readiness.md).

## Execution rules

- Roles are narrow contracts, not personas. Give each worker immutable inputs and one lens.
- Review workers are read-only. They return exact evidence and do not fix findings.
- For a release-readiness request, launch the complete Phase-1 review team with `../../scripts/jsdlc team --repo <repo> --candidate <candidate> --objective <objective>`. Do not manually substitute a single generic QA pass. The command runs QA Architect, QA Executor, Specialist Reviewer, and Independent Verifier in separate ephemeral read-only sessions against one frozen digest. It reports `OBSERVED_DISTINCT_SUBPROCESSES`, but remains `MANAGED_SEPARATE_PASSES` because CLI output cannot attest worker identity.
- Use `worker` only for an explicitly bounded extra pass. Its receipt has `assuranceEffect: EVIDENCE_ONLY` and cannot independently establish assurance.
- The orchestrator adjudicates findings. A corrector receives only accepted finding IDs and bounded targets.
- Re-run affected checks after correction. A fresh verifier checks the exact final candidate.
- Scripts establish deterministic facts; agents supply judgment.
- Tool failure, stale evidence, and applicable but untestable risk produce `INCONCLUSIVE` or `BLOCKED`, never a pass.
- Never deploy, publish, tag, release, spend money, access secrets, or perform destructive operations without authority supplied by the user and allowed by repository policy.

## Completion

Report the exact candidate, checks executed, findings and dispositions, assurance mode, limitations, and one terminal status. Accept a `READY` result only from the live `team` aggregation when all four required roles are clean and the Independent Verifier supplies nonblank `PASS` evidence for every release-risk domain. `NOT_APPLICABLE` remains `INCONCLUSIVE` until adjudication exists. Any finding produces `NOT_READY`; missing or blocked evidence produces `INCONCLUSIVE`. A readiness verdict never authorizes deployment, publishing, tagging, or release.
