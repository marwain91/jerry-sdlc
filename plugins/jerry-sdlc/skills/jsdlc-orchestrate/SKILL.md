---
name: jsdlc-orchestrate
description: Deliver software changes with proportionate planning, checks, and review, or assess explicit release readiness. Use for implementation, fixes, consequential diagnosis, code review, and product or UX design. Exclude small factual questions and read-only explanations without a delivery or design assessment.
---

# Jerry SDLC orchestrator

Everyday delivery is lightweight by default. Select the smallest workflow that resolves the user's outcome with credible evidence. No project initialization, CLI record, or formal verdict is required.

## Everyday work

Read [delivery-workflows.md](references/delivery-workflows.md). Follow repository execution rules, including containers. Identify acceptance criteria, actual risk, useful behavior to retain, and a bounded delivery slice. A slice is an observable outcome, not the entire product backlog.

Small mechanical edits need the edit and a relevant check. Ordinary features and fixes need one implementation owner, meaningful checks, and one bounded read-only review. The owner can plan, diagnose, run QA, and reconcile results. Add a specialist only for a named material risk, design uncertainty, or missing coverage. Combine code and focused UX review when one reviewer can cover both. Parallelize distinct lenses when justified and available; reviewers remain read-only.

For changed human-facing behavior, read [product-design-ux.md](references/product-design-ux.md). Experience assessment and actual interaction checks remain required; experience scope does not mandate extra agents. New journeys need a brief before implementation; the owner can write it.

Use one initial review and one targeted correction review per slice. Further review needs a named material blocker. Queue optional polish separately. Preserve authorization and continue necessary work through bounded slices and checkpoints. Do not declare success with unmet acceptance criteria.

For costly or multi-step work, read [efficiency.md](references/efficiency.md). Probe shared prerequisites once before delegation, set slice budgets, and wait on exact job identities with bounded automated monitoring. Measure worker-inclusive costs when available; missing measurements are unknown. Local ledgers cannot enforce runtime token caps.

## Optional recorded evidence

The bundled CLI is `../../scripts/jsdlc` relative to this file. Prefix commands in references with that path and run them in the repository-authorized environment.

Read [recorded-delivery.md](references/recorded-delivery.md) only when a durable local record is requested or useful. New records default to the lightweight profile; full retains the larger contract. Reuse unchanged deterministic checks through explicit dependency and runtime identity records. Never reuse reviewer conclusions across changed candidates. Scope and dependency completeness remain caller-declared.

Do not run doctor, inventory scans, or persistence commands as routine completion ceremony. Run doctor when managed workers are needed. Inspect coexistence when a competitor is observed, configuration changes, or formal release work requires it; use the release guide's collision procedure.

## Formal release readiness

Only an explicit readiness assessment uses [release-readiness.md](references/release-readiness.md), its control contract, doctor, and formal verifier. Everyday completion never changes that verdict. Generic HIGH-risk recorded delivery still lacks specialist persistence support; perform necessary specialist work natively and report the limit. Never lower risk to obtain a record.

## Handoff and messages

Report changes, observed checks, remaining criteria, and relevant coverage limits. Distinct passes do not establish attested independence. Local evidence is LOCAL_UNATTESTED. No result authorizes deployment, publication, tagging, release, spending, secret access, or destructive actions. Incident repository evidence does not establish production recovery.

Planning and review requests retain their authorized scope. Preserve the latest user decisions across handoffs. Ask for a new thread only after actually updating or reinstalling the active plugin.

Write English user messages and authored report text in ASD-STE100 Simplified Technical English. Use short, direct sentences and consistent terms; preserve exact technical names and quotations. Do not claim verified conformance without checking the official standard.
