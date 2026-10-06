# Product design and UX

Assess experience separately from engineering risk. Inspect the requested outcome and actual affected surface. File names are clues, not proof that a full design team is needed.

| Experience | Scope | Lightweight responsibility |
|---|---|---|
| none | Internal behavior or mechanical copy with unchanged meaning/layout | Owner notes applicability. |
| focused | Bounded interface, consequential copy, or accessibility change; existing-experience review | Owner sets a few criteria; the code reviewer can also cover UX. |
| full | New screens/products or substantially changed journeys | Owner writes a brief before implementation; reviewer checks the resulting journey. Add a designer for a named uncertainty. |

Human-facing surfaces include GUI, operator tools, CLI output, reports, and transactional messages. Exposed backend errors or recovery can affect experience. A pure internal/API implementation can remain none. UI presence does not raise engineering risk to HIGH.

A button alignment fix needs focused checks, not discovery. Explicit design reviews remain read-only; planning-only requests end with a brief and open decisions. During incidents, immediate authorized mitigation takes priority; applicable experience checks remain part of bounded verification. Do not expand scope into a redesign.

## Brief and retained behavior

Inspect established components, constraints, and current journeys. Reuse the existing design system and user-chosen aesthetics. For focused work, a short task/state/acceptance note is sufficient. For full work, capture the intended user and task, entry-to-completion journey, relevant interaction states, information priorities, and observable criteria. Label assumptions; do not invent research.

Cover only relevant loading, empty, success, error, permission, cancellation, and recovery states. Include applicable keyboard/focus behavior, accessible names, readable content, and representative supported viewports. Name useful existing behavior to retain, such as history, navigation state, or compact readable rows.

The brief can stay in the handoff. Add a repository artifact or visual only when it resolves a consequential decision or helps maintain the product. Ordinary design choices do not add approval gates or authorize user outreach, analytics changes, paid tools, or publishing.

Use short, consistent English messages following the skill's ASD-STE100 guidance. Preserve required product/technical terms. Explain what happened and relevant next actions. Do not claim verified standard conformance without checking it.

## Observe and review

Exercise the actual changed surface and relevant failure/recovery path in the authorized runtime. GUI work needs representative rendering and changed interactions; CLI/report work needs actual output and reading/interaction sequence. Source inspection, screenshots, and a passing build establish limited coverage.

Link each essential criterion to the actual state/input, expected and observed result, and an artifact/check reference. Do not infer an observation from source or invent device/accessibility coverage. Missing essential access or interaction evidence remains an explicit gap. Review task completion, not personal visual taste. Optional cosmetic polish should not trigger repeated delivery rounds.

One reviewer can cover code and focused UX when qualified and equipped. Add [Product Designer](../../../roles/product-designer.md) or [UX Reviewer](../../../roles/ux-reviewer.md) for a named uncertainty or coverage gap. Roles are responsibilities; separate agents do not establish attested independence. Apply findings through the one implementation owner and use targeted correction review.

## Optional records

Carry the selected experience into delivery-start. The lightweight profile covers brief and experience criteria in the owner/reviewer evidence; it does not automatically add design roles. Explicitly added design roles need reports. The full profile retains mandatory focused/full role additions. Read [recorded-delivery.md](recorded-delivery.md) only when persisting evidence.

Do not downgrade scope to obtain completion. Experimental experience checks do not activate release UX packs, prove user satisfaction, or establish independent assurance.
