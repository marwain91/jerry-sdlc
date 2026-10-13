# Delegation handoffs

Delegate only when a separate pass adds coverage the owner cannot give. A subagent does not see your context. Give it everything it needs in one message, and keep reviewers read-only.

## Reviewer

Send:

- the exact diff or commit range and how to view it
- the acceptance note (outcome, original failure path, behavior to keep)
- the checks you ran and their actual results, with the canonical commands and runtime
- the user's latest decisions and constraints that affect the change
- one bounded question, for example "Does this fix the reload path without breaking offline drafts?"

Ask for: findings ranked by impact, each with location, evidence, severity, and a concrete fix; and explicitly "no findings" when clean. Correctness, regressions, security, and missing tests come first. Focused UX is in scope when it applies. Polish goes in a separate, optional list.

For the re-review, send only the correction diff and the findings it addresses.

## Cross-model reviewer

A reviewer from another model family catches errors that a same-model reviewer shares with you. Use it for features and risky fixes when the other CLI is already installed and signed in. Probe it once (`codex --version` or `claude --version`); never install it, sign it in, or change its configuration. If it is unavailable, use a same-model reviewer and say so in the handoff.

Put the reviewer handoff above into the prompt. Keep the reviewer read-only:

- From Claude Code, review with Codex: `codex exec -s read-only --ephemeral -o <findings-file> -` (with the findings file outside the repository) and pass the handoff, including the diff command, on stdin. Do not use `codex review`: it cannot take a custom prompt together with a diff selector.
- From Codex, review with Claude Code: `claude -p --permission-mode plan` and pass the handoff, including the diff command, on stdin.

Both commands need network access. If your sandbox blocks the call, use a same-model reviewer and report it; do not widen the sandbox yourself.

Treat its output like any reviewer's findings: verify each one before you act on it. Count it as the one initial review, not an extra round.

## Specialist

Same as the reviewer, but name the single risk and the question, for example "Can an unauthenticated caller reach the new export endpoint?" Ask for one result for that risk only.

## Debugger

Send the reported path (entry point, state, actions, expected, actual), what you have already ruled out, and whether a fix is authorized. Ask for the supported cause, the evidence, open hypotheses, and the next check that would distinguish them.

## Designer

Send the user's objective, the affected surfaces, and the constraints (design system, brand, scope). Ask for the brief from [ux.md](ux.md) and one recommendation for each open decision. The designer works read-only; the owner implements.
