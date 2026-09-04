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
- Before the team run, an already-authorized deterministic command may be captured with `../../scripts/jsdlc check --repo <repo> --candidate <candidate> --id <stable-id> --domains <claimed-domains> --authorized -- <argv...>`. This is a fully privileged local command runner, not a sandbox or permission boundary: do not invoke it merely because the workflow wants evidence. Its records are visibly `LOCAL_UNATTESTED` and cannot satisfy readiness until an attested adapter and policy-defined check specs exist.
- If a check process is interrupted, do not delete state files manually. After confirming the exact command should no longer be running, use `recover-check --repo <repo> --candidate <candidate> --id <stable-id> --authorized`; it refuses while the owner or contained command process group is live and never overwrites committed evidence. If interruption occurred before the process-group identity became durable, recovery deliberately refuses and the safe path is a fresh run.
- Launch the complete review team with `../../scripts/jsdlc team --repo <repo> --candidate <candidate> --objective <objective>`. Do not manually substitute a single generic QA pass. PASS domain claims cite relevant check IDs, but locally self-labelled domains do not establish coverage. Reports and receipts are persisted outside the repository and bound to the candidate, checks, and frozen contracts.
- Use `worker` only for an explicitly bounded extra pass. Its receipt has `assuranceEffect: EVIDENCE_ONLY` and cannot independently establish assurance.
- The orchestrator adjudicates findings. A corrector receives only accepted finding IDs and bounded targets.
- Re-run affected checks after correction. A fresh verifier checks the exact final candidate.
- Scripts establish deterministic facts; agents supply judgment.
- Tool failure, stale evidence, and applicable but untestable risk produce `INCONCLUSIVE` or `BLOCKED`, never a pass.
- Never deploy, publish, tag, release, spend money, access secrets, or perform destructive operations without authority supplied by the user and allowed by repository policy.

## Completion

Run `../../scripts/jsdlc verify --repo <repo> --candidate <candidate>` before reporting the terminal status. It reproduces the latest persisted verdict against the current candidate, checked evidence, reports, receipts, and contracts. `READY` additionally requires `MANAGED_INDEPENDENT`; clean separate passes remain `INCONCLUSIVE`. `NOT_APPLICABLE` remains `INCONCLUSIVE` until adjudication exists. Any finding produces `NOT_READY`; missing, failed, stale, or blocked evidence produces `INCONCLUSIVE` or `BLOCKED`. A readiness verdict never authorizes deployment, publishing, tagging, or release.
