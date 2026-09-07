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

## Shared handoff

Inspect repository instructions first. State the selected workflow and team briefly. Give reviewers the objective, relevant requirements, exact changed surface, test evidence, and unresolved assumptions. When host policy permits, use separate subagents for distinct non-trivial read-only roles and run independent lenses in parallel. Review roles stay read-only and report evidence before recommendations. Keep one clear owner for implementation and corrections.

Run project checks only through the execution environment authorized by repository instructions. A claimed pass must come from observed command output. If a check cannot run, name the missing evidence instead of treating review as proof.

## Workflow exits

- `trivial-change`: the requested edit is present and a proportionate verification passes.
- `bug-diagnosis`: the cause is supported by evidence; no fix is made unless requested.
- `bug-fix`: the failure is reproduced or otherwise evidenced, the bounded fix is implemented, regression coverage passes, and review finds no unresolved blocker.
- `feature`: acceptance criteria are met, relevant tests pass, code review is clean, and user-facing or operational documentation is updated when needed.
- `pr-review`: findings are ordered by severity with exact locations and evidence; if none exist, state residual test or coverage limits.
- `incident`: impact and timeline are recorded, mitigation is verified, destructive or production actions retain their normal authorization gates, and follow-up work is explicit.

Do not call any everyday reviewer independent unless the runtime actually provides independent identity and isolation. Say `separate review pass` when that is all that is known.

After implementation is frozen—or immediately for read-only diagnosis and PR review—start the matching everyday delivery record, persist one report for every selected role, and run `delivery-verify`. `COMPLETE` means the everyday contract is complete for that exact repository content only. `ISSUES`, `INCOMPLETE`, and `BLOCKED` are not release verdicts; all everyday outcomes have no release-readiness effect.

## Role contracts

Load only the contracts selected for the workflow: [Delivery Planner](../../../roles/delivery-planner.md), [Implementer](../../../roles/implementer.md), [Debugger](../../../roles/debugger.md), [Code Reviewer](../../../roles/code-reviewer.md), [QA Executor](../../../roles/qa-executor.md), [Verifier](../../../roles/verifier.md), or [Incident Commander](../../../roles/incident-commander.md).
