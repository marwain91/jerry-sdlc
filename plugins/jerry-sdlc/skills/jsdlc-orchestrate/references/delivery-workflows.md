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

The table gives base engineering roles. Assess experience independently for every task using [product-design-ux.md](product-design-ux.md): `none` retains the base team, `focused` adds UX Reviewer, and `full` adds Product Designer and UX Reviewer. User-facing changes require the applicable experience pass even when engineering risk is LOW. New or substantially changed journeys receive product design before implementation; bounded interface fixes receive focused review. Design-only requests retain their read-only or planning scope.

Use `classify --request <request> --files <comma-separated-paths>` for the deterministic workflow/risk/experience proposal, then assess actual scope and raise risk where judgment requires. Runtime risk labels are LOW, NORMAL, and HIGH. Native teams may scale down as described above; persisted completion always requires the complete role set from `roles --workflow <workflow> --experience <none|focused|full>`.

## Shared handoff

Inspect repository instructions first. State the selected workflow and team briefly. Give reviewers the objective, relevant requirements, exact changed surface, test evidence, and unresolved assumptions. When host policy permits, use separate subagents for distinct non-trivial read-only roles and run independent lenses in parallel. Review roles stay read-only and report evidence before recommendations. Keep one clear owner for implementation and corrections.

Keep the handoff proportionate; a few lines suffice for small tasks. Carry these facts across delegation, continuation, and compaction:

- Intended outcome and acceptance criteria; for fixes, the reported starting state, action sequence, and symptom; useful behavior to retain.
- Already-authorized actions, exclusions and holds, and the latest user decision. Preserve authorization within its scope instead of asking again. A generic “continue” does not lift a hold or authorize a release; clarify a consequential ambiguity rather than discarding a condition.
- Candidate and changed scope, existing unrelated work, relevant evidence, unresolved requirements, and next action. Mark missing candidate identity or evidence as unknown rather than omitting it or inventing it. New messages update the work without silently dropping earlier requirements.
- The bounded question for each reviewer and the missing evidence that would resolve it.

Run project checks only through the execution environment authorized by repository instructions. A claimed pass must come from observed command output. If a check cannot run, name the missing evidence instead of treating review as proof.

Before executing checks, inspect existing project configuration to identify the permitted runtime/container, canonical local and CI commands, affected consumers, and any authorized deployment verification route. Include the runtime and check commands in the QA handoff, or identify what remains to be discovered; record material environment differences. Do not install host dependencies or invent an execution path merely because a familiar command is unavailable.

For everyday work without a QA Architect assignment, the orchestrator supplies the acceptance criteria and test strategy to QA Executor, informed by Delivery Planner or Debugger. Add a dedicated QA Architect pass when complexity warrants it.

Use targeted checks while editing a coherent batch, then required checks on the frozen candidate. Repeat work when a relevant change, failure, or unresolved question justifies it; diagnose an unchanged tool failure before retrying the same action. Preserve failed results and explain any test correction rather than weakening assertions to obtain a pass. This proportionality does not waive required roles, the verifier's additional check execution, or candidate binding: earlier-candidate evidence cannot verify a new candidate.

## Workflow exits

- `trivial-change`: the requested edit is present and a proportionate verification passes.
- `bug-diagnosis`: the cause is supported by evidence; no fix is made unless requested.
- `bug-fix`: the failure is reproduced or otherwise evidenced, the bounded fix is implemented, regression coverage passes, and review finds no unresolved blocker.
- `feature`: acceptance criteria are met, relevant tests pass, code review is clean, applicable product design and UX criteria have observed coverage with no unresolved findings, and user-facing or operational documentation is updated when needed.
- `pr-review`: findings are ordered by severity with exact locations and evidence; if none exist, state residual test or coverage limits.
- `incident`: impact, timeline, supported findings, remaining uncertainty, and follow-up work are recorded. For investigation-only requests, report the investigation outcome without implying recovery. When mitigation is authorized and performed, verify its production effect before claiming recovery; production and destructive actions retain their normal authorization gates.

Do not call any everyday reviewer independent unless the runtime actually provides independent identity and isolation. Say `separate review pass` when that is all that is known.

Before final handoff, the orchestrator reconciles tracked tasks with their actual acceptance criteria and reads back saved checklist/status changes. Close implementation-only tasks when their criteria are met; do not make unrelated deployment a new requirement. Conversely, keep required deployment, release artifacts, or external approval outstanding until evidenced. Report applicable milestones separately and name any remaining action/owner. Updating a task tracker or executing a release still requires the corresponding authorization; a reviewer stays read-only.

When the user separately authorizes release execution, follow the project's release procedure and identify required artifacts per target: previous published version, target version, applicable localized notes/metadata, and distribution state. Compare against each target's own published baseline; platforms may have different previous versions. Verify those requirements before claiming the release request finished. Formal `release-readiness` only assesses its candidate and does not execute or authorize any of these actions.

After implementation is frozen—or immediately for read-only diagnosis and PR review—start the matching everyday delivery record, persist one report for every selected role, and run `delivery-verify`. `COMPLETE` means the everyday contract is complete for that exact repository content only. `ISSUES`, `INCOMPLETE`, and `BLOCKED` are not release verdicts; all everyday outcomes have no release-readiness effect.

## Persisted everyday evidence

Prefix each command with the bundled CLI path defined in SKILL.md and use the repository-authorized execution environment.

1. Plan and implement first. Freeze the candidate after edits, before QA/review. For read-only diagnosis or PR review, freeze at the start. Use a commit SHA or explicit working-tree label.
2. Inspect `delivery-status --repo <repo> --candidate <candidate>`. Resume only a matching, non-stale ACTIVE run with the selected experience scope. Missing state or a terminal run permits `delivery-start --repo <repo> --candidate <candidate> --workflow <workflow> --risk <risk> --experience <none|focused|full>`; do not hide other status errors. An active collision must be resolved first. Stale active runs need explicit `delivery-cancel` with their recorded candidate before a fresh run.
3. Capture already-authorized commands with `delivery-check --repo <repo> --candidate <candidate> --id <id> --authorized -- <argv...>`. It executes with full local privileges; it is not a sandbox. Interrupted reservations use `delivery-recover-check` with the same identity flags and `--authorized`; recovery refuses live or indeterminate processes and committed evidence.
4. Record every role required by `roles --workflow <workflow> --experience <none|focused|full>` using `delivery-record --repo <repo> --candidate <candidate> --role <role> --file <report.json> --producer-mode <SELF_REVIEW|SEPARATE_PASS>`. Reports follow `schemas/delivery-report-input.schema.json`. Native risk-scaled teams may be smaller, but persisted completion requires the full CLI contract. Planning/design/implementation reports describe work on the frozen candidate; reviewers remain read-only.
5. Clean QA Executor, UX Reviewer, and Verifier reports cite successful unchanged-candidate check IDs. UX Reviewer may reuse QA checks when they support actual experience observations. When QA Executor and Verifier both apply, Verifier must cite at least one additional check execution. A different ID alone is not proof of a distinct agent. All recorded checks are verified, even uncited ones; a successful retry does not dismiss an earlier failure.
6. Run `delivery-verify --repo <repo> --candidate <candidate>`. Outcomes are COMPLETE, ISSUES, INCOMPLETE, or BLOCKED, always with `releaseReadinessEffect: NONE`. Missing reports leave the run ACTIVE so recording can continue. Candidate or contract drift requires a fresh run; evidence cannot transfer to a changed candidate.

Reports cannot be replaced through the CLI. Finalization detects evidence changes while its local anchor remains intact; the same OS owner can rewrite both. Keep report files outside the frozen repository and exclude secrets. Generic HIGH-risk persistence requires specialist support that is not implemented; continue needed specialist work natively and report the persistence limitation. The incident exception records repository-side evidence only.

## Role contracts

Load only the contracts selected for the workflow: [Delivery Planner](../../../roles/delivery-planner.md), [Implementer](../../../roles/implementer.md), [Debugger](../../../roles/debugger.md), [Code Reviewer](../../../roles/code-reviewer.md), [QA Executor](../../../roles/qa-executor.md), [Verifier](../../../roles/verifier.md), or [Incident Commander](../../../roles/incident-commander.md).

For the selected experience scope also load [Product Designer](../../../roles/product-designer.md) and/or [UX Reviewer](../../../roles/ux-reviewer.md) as directed by the product design guide.
