# Product design and UX

Human-facing surfaces include GUIs, operator tools, CLI output, reports, transactional messages, and exposed error or recovery text. A pure internal or API change has no experience scope. A UI change does not by itself raise engineering risk.

## Scope

| Scope | When | What to do |
|---|---|---|
| none | Internal behavior, or copy with unchanged meaning and layout | Nothing extra. |
| focused | A bounded interface, copy, or accessibility change | A short note: task, affected states, acceptance criteria. The code reviewer covers UX. |
| full | New screens or products, or substantially changed journeys | Write a brief before implementing. Add a designer pass only for a named uncertainty. |

A button alignment fix is focused, not a discovery project. Do not expand scope into a redesign.

## Brief (full scope)

- intended user and task; current friction, with evidence if any
- entry-to-completion journey
- relevant states: loading, empty, success, error, permission, cancellation, recovery
- information priority and copy
- what to reuse from the existing design system and components
- useful existing behavior to keep (history, navigation state, dense readable rows)
- observable acceptance criteria

Label assumptions. Do not invent research, personas, metrics, or user approval. Consider alternatives only when they materially change effort, discoverability, or recovery; recommend one and state the tradeoff. The brief can live in the handoff; add a repository file or mockup only when it settles a consequential decision.

## Check it for real

Exercise the actual changed surface in the authorized runtime, including the relevant failure and recovery path. GUI work needs real rendering and real interactions at representative viewports; CLI and report work needs actual output.

- Source inspection and a passing build establish neither appearance nor usability.
- A screenshot supports appearance, not interaction.
- Check keyboard and focus behavior, accessible names, and readable content where applicable. Separate measured accessibility checks from manual observation; do not claim blanket compliance.
- If you cannot render or interact, say so. That is a gap, not a pass.

Review against the user's task, not personal taste. Report task-blocking or misleading behavior before cosmetic issues.
