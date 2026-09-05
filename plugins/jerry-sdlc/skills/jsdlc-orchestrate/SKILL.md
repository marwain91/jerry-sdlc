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
4. For delivery work, derive a stable candidate label (prefer the exact commit SHA for a clean candidate; use an explicit working-tree label otherwise). Run `../../scripts/jsdlc status --repo <repo> --candidate <candidate>` first. Resume only when it returns a matching, non-terminal run. When it reports that no state exists, run `../../scripts/jsdlc start --repo <repo> --candidate <candidate> --workflow <workflow> --assurance <doctor-outcome>`. Never replace a mismatched active run; report the collision. Starting this external run record is automatic workflow bootstrap, not repository initialization and needs no user setup.
5. If `doctor` returns `UNAVAILABLE`, stop before starting or launching workers and provide its diagnostics. Otherwise briefly tell the user the workflow, risk, assurance mode, and roles being used.

If the runtime or user reports another broad orchestrator, record that caller-supplied evidence using `schemas/collision-input.schema.json` and run `../../scripts/jsdlc resolve-collision --file <evidence.json>` before starting. The command only resolves precedence; it does not discover plugins or launch work. Follow an explicit user choice, resume only the same active Jerry run, and stop at `OWNER_GATE` for every unresolved collision.

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
- The orchestrator proposes adjudication, but explicit local decisions are recorded only with `adjudicate --repo <repo> --candidate <candidate> --file <decision.json> --authorized`. Bind the file to the exact returned team-evidence digest. Accepted findings remain blocking; rejected findings require rationale and successful checked evidence. A `NOT_APPLICABLE` decision additionally requires successful check IDs relevant to that domain. Adjudication never upgrades assurance.
- A corrector receives only accepted finding IDs and bounded targets. Before editing, record the separate authorization with `authorize-correction --repo <repo> --candidate <candidate> --file <correction.json> --authorized`; its input binds the team digest, accepted finding IDs, and exact paths (use a trailing `/` only for an authorized subtree). Do not treat adjudication alone as permission to edit.
- Local mode does not sandbox the corrector: it rejects repositories containing symlinks and audits scope only after editing. Do not claim it prevented a same-user process from making and hiding external changes. After editing, run `finish-correction --repo <repo> --candidate <old> --new-candidate <new> --authorization <digest> --authorized`. It rejects visible out-of-scope or empty repository changes, caps the lineage at two cycles, and creates a fresh baselined run. Re-run affected checks and the complete team; the prior evidence cannot verify the new candidate.
- Scripts establish deterministic facts; agents supply judgment.
- Tool failure, stale evidence, and applicable but untestable risk produce `INCONCLUSIVE` or `BLOCKED`, never a pass.
- Never deploy, publish, tag, release, spend money, access secrets, or perform destructive operations without authority supplied by the user and allowed by repository policy.

## Completion

Run `../../scripts/jsdlc verify --repo <repo> --candidate <candidate>` before reporting the terminal status. It reproduces the latest persisted verdict against the current candidate, checked evidence, reports, receipts, and contracts. `READY` additionally requires `MANAGED_INDEPENDENT`; clean separate passes remain `INCONCLUSIVE`. `NOT_APPLICABLE` remains `INCONCLUSIVE` until adjudication exists. Any finding produces `NOT_READY`; missing, failed, stale, or blocked evidence produces `INCONCLUSIVE` or `BLOCKED`. A readiness verdict never authorizes deployment, publishing, tagging, or release.
