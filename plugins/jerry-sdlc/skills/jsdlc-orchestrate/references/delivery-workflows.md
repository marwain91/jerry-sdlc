# Everyday delivery

Everyday delivery is experimental and lightweight by default. Roles are responsibilities; they do not require one agent per responsibility. CLI persistence is optional. An ordinary engineering handoff needs no JSON reports or COMPLETE label.

## Routing

| Workflow | Default work |
|---|---|
| trivial-change | One owner edits and runs a relevant check. No delegation needed. |
| bug-diagnosis | One debugger establishes the cause. Add review for a consequential or uncertain conclusion. No unauthorized fix. |
| bug-fix | One owner evidences the failure, implements the fix, and runs regression checks; one reviewer checks changed behavior. |
| feature | One owner plans, implements, and tests a bounded outcome; one reviewer covers correctness and applicable UX. |
| pr-review | One read-only reviewer returns findings and test limits. Execute extra checks only for a coverage gap. |
| incident | Coordinate impact, diagnosis, and authorized mitigation; verify production effect before claiming recovery. |
| release-readiness | Explicit formal assessment; use the separate release contract. |

Authentication, authorization, secrets, payments, destructive data changes, migrations, public APIs, concurrency, and production infrastructure need applicable specialist attention. Name the risk and bounded review question. Do not add every specialist because one risk exists. Engineering risk and experience scope are separate.

The optional classify command proposes workflow/risk/experience. Inspect actual scope and raise risk when needed. File extensions alone do not justify a redesign. `roles --workflow feature` returns the lightweight record profile; `--profile full` returns the larger profile. Native teams can combine responsibilities; requested records follow their selected contract.

## Work and checks

Keep a short acceptance note: intended outcome, original failure path, behavior to retain, changed surface, and relevant checks. Inspect existing work and project runtime/CI configuration before edits. Use an isolated worktree for overlapping changes; preserve unrelated files. Carry canonical commands and runtime into reviewer handoffs.

The owner supplies test strategy; add QA architecture only when uncertainty warrants it. Run targeted checks while editing and required project checks on the final candidate. Give the reviewer the exact diff, acceptance note, observed results, and a bounded question. Reviewers remain read-only.

Use one initial review and one targeted correction review per slice. Further passes must resolve a named material blocker. Optional cosmetic or readability suggestions do not restart completed delivery. Apply valid correctness findings; do not weaken assertions or discard failures to obtain a pass.

Rerun evidence after relevant dependency/environment changes, failures, or unresolved questions. A verifier reconciles criteria and evidence; it need not duplicate an adequate check. Native lightweight work can reuse observed evidence with its exact identity and limits. Recorded reuse follows [recorded-delivery.md](recorded-delivery.md). Old evidence never covers changed behavior.

For costly work, set per-slice limits and use [efficiency.md](efficiency.md). Probe shared uncertain prerequisites before delegation. Diagnose an unchanged tool failure before retrying. Use completion signals or bounded polling for live CI; preserve job identity on timeout.

## Completion

Acceptance criteria and relevant observed checks determine completion. Fixes cover the original symptom and retained behavior. Interfaces need actual changed interactions and applicable recovery checks. Source inspection and screenshots have limited coverage.

When task updates are authorized, reconcile saved acceptance criteria and read back checklist/status changes. Do not invent deployment for an implementation-only task. Requested deployment or release remains outstanding until verified. Continuation preserves authorization and holds; it does not authorize a release.

Separately authorized release execution follows the project's procedure. Verify target versions, required artifacts, localized metadata, distribution state, and each target's published baseline. Readiness assessment does not execute a release.

Report engineering results before relevant assurance limits. Missing essential coverage remains an explicit gap. Separate passes are not attested independence. Repository incident completion does not prove production recovery.

Load only needed contracts: [Implementer](../../../roles/implementer.md), [Code Reviewer](../../../roles/code-reviewer.md), [Debugger](../../../roles/debugger.md), [Verifier](../../../roles/verifier.md), [QA Executor](../../../roles/qa-executor.md), [Delivery Planner](../../../roles/delivery-planner.md), or [Incident Commander](../../../roles/incident-commander.md). For experience work read [product-design-ux.md](product-design-ux.md).
