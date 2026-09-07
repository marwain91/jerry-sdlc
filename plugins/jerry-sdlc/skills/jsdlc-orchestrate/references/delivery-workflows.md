# Everyday delivery workflows

These workflows are experimental Codex-native coordination patterns. At the review boundary they persist candidate-bound local role reports through the dedicated `delivery-*` CLI namespace. That evidence is locally owned and unattested. It never produces or affects the formal release verdict provided by `release-readiness`.

## Routing

| Workflow | Use when | Default team |
|---|---|---|
| `trivial-change` | A clearly local typo, copy, formatting, or mechanical edit with negligible behavioral risk | Implementer, then Verifier |
| `bug-diagnosis` | The user asks for cause or investigation without authorizing a fix | Debugger, then Code Reviewer when the conclusion is consequential |
| `bug-fix` | Reproducing and correcting broken behavior | Debugger, Implementer, QA Executor, Code Reviewer, Verifier |
| `feature` | Adding or changing behavior | Delivery Planner, Implementer, QA Executor, Code Reviewer, Verifier |
| `pr-review` | Reviewing a PR, branch, commit, or diff without changing it | Code Reviewer; add QA Executor and Verifier for behavioral or high-risk changes |
| `incident` | Outage, active production degradation, security incident, or urgent operational investigation | Incident Commander, Debugger, QA Executor, Verifier |
| `release-readiness` | Deciding whether an immutable candidate should ship | Use the formal release workflow and its complete release team |

Risk overrides labels. Authentication, authorization, secrets, payments, destructive data changes, migrations, public APIs, concurrency, production infrastructure, or broad cross-component changes require specialist review even when the request sounds small. A large documentation-only change does not automatically need a feature team.

Use `classify --request <request> --files <comma-separated-paths>` for the deterministic workflow/risk proposal, then raise risk where judgment requires. Runtime risk labels are LOW, NORMAL, and HIGH. Native teams may scale down as described above; persisted completion always requires the complete role set from `roles --workflow <workflow>`.

## Shared handoff

Inspect repository instructions first. State the selected workflow and team briefly. Give reviewers the objective, relevant requirements, exact changed surface, test evidence, and unresolved assumptions. When host policy permits, use separate subagents for distinct non-trivial read-only roles and run independent lenses in parallel. Review roles stay read-only and report evidence before recommendations. Keep one clear owner for implementation and corrections.

Run project checks only through the execution environment authorized by repository instructions. A claimed pass must come from observed command output. If a check cannot run, name the missing evidence instead of treating review as proof.

For everyday work without a QA Architect assignment, the orchestrator supplies the acceptance criteria and test strategy to QA Executor, informed by Delivery Planner or Debugger. Add a dedicated QA Architect pass when complexity warrants it.

## Workflow exits

- `trivial-change`: the requested edit is present and a proportionate verification passes.
- `bug-diagnosis`: the cause is supported by evidence; no fix is made unless requested.
- `bug-fix`: the failure is reproduced or otherwise evidenced, the bounded fix is implemented, regression coverage passes, and review finds no unresolved blocker.
- `feature`: acceptance criteria are met, relevant tests pass, code review is clean, and user-facing or operational documentation is updated when needed.
- `pr-review`: findings are ordered by severity with exact locations and evidence; if none exist, state residual test or coverage limits.
- `incident`: impact, timeline, supported findings, remaining uncertainty, and follow-up work are recorded. For investigation-only requests, report the investigation outcome without implying recovery. When mitigation is authorized and performed, verify its production effect before claiming recovery; production and destructive actions retain their normal authorization gates.

Do not call any everyday reviewer independent unless the runtime actually provides independent identity and isolation. Say `separate review pass` when that is all that is known.

After implementation is frozen—or immediately for read-only diagnosis and PR review—start the matching everyday delivery record, persist one report for every selected role, and run `delivery-verify`. `COMPLETE` means the everyday contract is complete for that exact repository content only. `ISSUES`, `INCOMPLETE`, and `BLOCKED` are not release verdicts; all everyday outcomes have no release-readiness effect.

## Persisted everyday evidence

Prefix each command with the bundled CLI path defined in SKILL.md and use the repository-authorized execution environment.

1. Plan and implement first. Freeze the candidate after edits, before QA/review. For read-only diagnosis or PR review, freeze at the start. Use a commit SHA or explicit working-tree label.
2. Inspect `delivery-status --repo <repo> --candidate <candidate>`. Resume only a matching, non-stale ACTIVE run. Missing state or a terminal run permits `delivery-start --repo <repo> --candidate <candidate> --workflow <workflow> --risk <risk>`; do not hide other status errors. An active collision must be resolved first. Stale active runs need explicit `delivery-cancel` with their recorded candidate before a fresh run.
3. Capture already-authorized commands with `delivery-check --repo <repo> --candidate <candidate> --id <id> --authorized -- <argv...>`. It executes with full local privileges; it is not a sandbox. Interrupted reservations use `delivery-recover-check` with the same identity flags and `--authorized`; recovery refuses live or indeterminate processes and committed evidence.
4. Record every role required by `roles --workflow <workflow>` using `delivery-record --repo <repo> --candidate <candidate> --role <role> --file <report.json> --producer-mode <SELF_REVIEW|SEPARATE_PASS>`. Reports follow `schemas/delivery-report-input.schema.json`. Native risk-scaled teams may be smaller, but persisted completion requires the full CLI contract. Planning/implementation reports describe work on the frozen candidate; reviewers remain read-only.
5. Clean QA Executor and Verifier reports cite successful unchanged-candidate check IDs. When both apply, Verifier must cite at least one additional check execution. A different ID alone is not proof of a distinct agent. All recorded checks are verified, even uncited ones; a successful retry does not dismiss an earlier failure.
6. Run `delivery-verify --repo <repo> --candidate <candidate>`. Outcomes are COMPLETE, ISSUES, INCOMPLETE, or BLOCKED, always with `releaseReadinessEffect: NONE`. Missing reports leave the run ACTIVE so recording can continue. Candidate or contract drift requires a fresh run; evidence cannot transfer to a changed candidate.

Reports cannot be replaced through the CLI. Finalization detects evidence changes while its local anchor remains intact; the same OS owner can rewrite both. Keep report files outside the frozen repository and exclude secrets. Generic HIGH-risk persistence requires specialist support that is not implemented; continue needed specialist work natively and report the persistence limitation. The incident exception records repository-side evidence only.

## Role contracts

Load only the contracts selected for the workflow: [Delivery Planner](../../../roles/delivery-planner.md), [Implementer](../../../roles/implementer.md), [Debugger](../../../roles/debugger.md), [Code Reviewer](../../../roles/code-reviewer.md), [QA Executor](../../../roles/qa-executor.md), [Verifier](../../../roles/verifier.md), or [Incident Commander](../../../roles/incident-commander.md).
