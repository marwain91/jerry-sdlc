# Product design and UX in everyday delivery

Engineering correctness and experience quality are separate questions. Select experience scope at intake by inspecting the requested outcome and affected surface, including existing screens or interactive output when available. Reassess when scope changes. File extensions and classifier recommendations are clues, not proof of applicability.

## Select experience scope

| Experience | Apply when | Required participation |
|---|---|---|
| `none` | No human-facing behavior changes: internal refactor, infrastructure, data pipeline, or a mechanical typo with no change in meaning or layout | Existing engineering roles; give a brief applicability rationale |
| `focused` | A bounded UI fix, layout/component change, consequential copy, accessibility correction, or review of an existing experience | UX Reviewer; orchestrator supplies a short task/state/acceptance note before edits |
| `full` | New screens or products; navigation, onboarding, forms, search, checkout, or other substantially changed journeys; an open-ended redesign | Product Designer before implementation and UX Reviewer after candidate freeze |

Human-facing surfaces include mobile/web/desktop UI, operator tools, CLI prompts and help, generated reports, and transactional messages. Backend changes can require UX work when they alter exposed errors, timing, progress, or recovery. A pure API implementation without a changed human-facing contract can stay `none`.

Select proportionately: a button alignment fix needs `focused`, not a discovery project. A changed action label that alters expectations is not a mechanical typo. Explicit user requests for design or UX review activate the corresponding pass even if no code changes. For review-only requests use `pr-review`; do not infer permission to implement. For planning-only work, provide the brief and criteria and stop at the authorized planning outcome; do not claim implemented delivery completion.

UI presence alone does not imply HIGH engineering risk. Payments, authorization, destructive actions, and other actual risks keep their normal escalation. During incidents, immediate authorized mitigation takes priority; assess affected experience in the bounded verification/follow-up without delaying urgent recovery for a redesign.

Report observed production recovery separately from repository completion. If selected UX evidence is still missing, keep the repository outcome incomplete even when service recovery is verified. An existing misleading message outside the mitigation's changed scope can be a named follow-up; a changed user-facing mitigation still needs its applicable UX checks. Do not widen emergency work into an unrequested redesign.

## Before implementation

Load [Product Designer](../../../roles/product-designer.md) for `full` and [UX Reviewer](../../../roles/ux-reviewer.md) for both applicable scopes. State the selected experience level and roles alongside workflow and risk. Delegate distinct read-only design/review passes when available and permitted; otherwise perform explicit passes and label them `SELF_REVIEW`. Role names never establish independent assurance.

Inspect existing components, patterns, brand constraints, and the current journey before proposing a replacement. The brief may live in the task handoff; add a repository artifact only when useful to ongoing project work. For `focused`, a few concrete criteria are enough. For `full`, establish:

- Who is trying to do what, where they enter, what successful completion looks like, and which problem the change addresses. Label assumptions; use supplied research or actual observations when available.
- The primary journey and relevant alternatives, with information hierarchy, action priorities, navigation, and user-facing copy. Explain the design decisions that affect task effort or comprehension.
- Applicable initial, loading, empty, success, validation-error, system-error, permission, cancellation, and recovery states. Select states that exist in this feature; explain important exclusions instead of manufacturing every state.
- Applicable keyboard and focus behavior, accessible names, readable contrast/content, responsive layout, and input modalities. Reuse the established design system; preserve user-chosen aesthetics.
- Observable acceptance criteria and how to check them. Example: after a rejected submission, entered values remain, the error identifies the affected field, and keyboard users can reach and correct it.

For both scopes, name the useful existing behavior to retain: for example, a history view, navigation state, or the number of readable rows in a compact panel. Separate correcting underlying data from redesigning how people use it. Judge success through the intended task: technically valid rendering is insufficient if the information needed for that task remains unreadable.

Use visual artifacts when they make a consequential decision reviewable. Prefer the project's established tools and components. If a user decision is essential, ask a concise question and continue independent work; ordinary design choices do not introduce an approval gate. Do not infer permission for user outreach, analytics changes, external publishing, or paid tools.

## During implementation and validation

The implementer receives the brief, state expectations, component reuse choices, and acceptance criteria along with engineering requirements. Update that handoff for material scope changes. After changes, freeze the candidate and have UX Reviewer inspect the real surface against those criteria. Review task completion and recovery as well as visual hierarchy, density, consistency, and copy.

For graphical interfaces, inspect rendered output at representative supported viewports and exercise the changed interactions. For CLI/message/report surfaces, inspect actual output and the relevant interaction or reading sequence. Use repository-authorized runtime environments and available browser/device tools. Do not install a new toolchain or expand the product merely to satisfy a checklist.

Evidence should be compact and traceable, for example:

| Criterion | Observation | Evidence | Result |
|---|---|---|---|
| Invalid submission preserves input and exposes a useful error | Keyboard submission with one invalid field; values retained, field error announced by the tested mechanism | Candidate-bound check ID plus trace/output path | Met, finding, or unverified |

Do not fabricate this observation from source code or a screenshot. Automated accessibility checks do not establish full accessibility, and heuristic review is not user research. Missing rendering or interaction access means partial coverage, not a clean UX verdict. Address authorized findings through the implementation owner, then freeze and review the changed candidate again under the delivery lifecycle.

## Persisted completion

Pass `--experience focused` or `--experience full` to both `roles` and `delivery-start`; use `none` for non-applicable work. The default `none` exists for CLI compatibility, not as permission to omit the intake assessment. The classifier only recommends scope: the orchestrator must inspect the actual task and pass the selected value. Report this caller-declared limit truthfully.

The selected roles are part of the frozen run. `focused` requires `ux-reviewer`; `full` also requires `product-designer`. Record their reports with `delivery-record` using the existing report schema. Include the brief/criteria and observations in `evidence`; essential untested criteria require `INCONCLUSIVE` or `BLOCKED`, and actual defects require `FINDINGS`. A clean UX report must cite successful candidate-bound checks. The verifier checks the criteria-to-evidence mapping as well as engineering results.

Changing experience scope after freezing requires cancelling the matching active run and starting a fresh candidate record. Do not silently downgrade experience to obtain `COMPLETE`. Backend-only work retains its base roles. These experimental everyday roles do not activate the gated release `ux-accessibility` pack, alter release readiness, or prove product-market fit, user satisfaction, or independent assurance.
