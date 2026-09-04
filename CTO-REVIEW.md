# CTO review — Jerry SDLC plan

## Executive assessment

The product direction is coherent: risk-scaled workflows, evidence-based verification, truthful independence labels, and explicit release gates are strong foundations. However, the plan currently treats several advisory Codex behaviors as enforceable runtime guarantees. It also makes release readiness the first deliverable before defining the state, permission, execution, and evaluation mechanisms needed to support credible release assurance.

## Blocking issues

1. **Automatic triggering is probabilistic, not guaranteed.** Codex implicitly selects a skill when a request matches its description; selection remains model-driven. The MVP cannot promise that ordinary language always triggers the correct workflow. Define measured trigger precision/recall, collision behavior, an explicit fallback command, and supported Codex surfaces.

2. **Role permissions are described but not technically enforced.** A read-only reviewer contract does not itself prevent writes. Reviewer and corrector isolation requires separate read-only execution contexts, worktrees or snapshots, tool allowlists, or truthful advisory labelling. The same applies to immutable inputs.

3. **The execution and state model is missing.** The plan does not specify how workflows are represented, resumed, versioned, transitioned, or recovered after context compaction, interruption, tool failure, or user steering. Define a state machine, run ID, candidate revision, artifact storage, failure semantics, and staleness rules.

4. **The CLI delivery model is unresolved.** The plan does not explain how `jsdlc` becomes executable across supported environments without initialization or dependency installation. Specify whether validators are bundled scripts, a platform-specific binary, an MCP service, or an optional separately installed CLI. `classify` is deterministic only if its rules and inputs are deterministic.

5. **Security and governance are insufficient for the claimed automation.** The orchestrator consumes untrusted repository content, diffs, test output, and existing instructions. Define instruction precedence, data boundaries, secret handling, permitted network and external actions, command provenance, audit events, denial behavior, and handling of malicious repository instructions. Release authorization must bind to an exact commit and artifact digest and expire after mutation.

6. **Release readiness is not credible with deferred risk domains.** Database migration and distributed-systems reliability review are postponed, yet the flagship claims general release readiness. Include minimum migration, deployment, rollback, dependency/supply-chain, observability, and reliability checks, or explicitly constrain the MVP to repositories without those risks.

7. **Acceptance and evaluation criteria are not falsifiable.** “Correct workflow,” “measured risk,” “better defect detection,” and “unacceptable growth” need thresholds. Define labelled datasets, independent ground truth, trigger confusion matrices, severity-weighted defect recall, false-positive rate, unauthorized-action rate, evidence-fabrication rate, latency/cost budgets, repeated trials, and pass/fail thresholds.

## Important improvements

- Reduce the MVP to one narrow release workflow plus PR review, with approximately five operational roles: orchestrator, implementer, QA, specialist reviewer, and verifier. Several proposed roles may be stages or lenses rather than distinct workers.
- Separate advisory, managed, and enforced modes and state exactly what each guarantees.
- Replace “when runtime capability permits” with a capability handshake and deterministic degradation matrix.
- Establish policy precedence among user instructions, `AGENTS.md`, optional Jerry policy, plugin policy, and platform security controls.
- Define cancellation, budgets, maximum correction cycles, duplicate findings, reviewer disagreement, flaky tests, partial coverage, and inconclusive outcomes.
- Version schemas, roles, workflows, and evidence together, recording their versions and the repository revision in every report.
- Distinguish source review from artifact verification: production verification requires an immutable build artifact and target environment, not merely a working tree.
- Add an activation smoke test. Zero-init can mean no repository mutation, but not zero capability discovery or diagnostics.
- Validate plugin installation separately from workflow quality.

## Verdict

**REVISE**

The concept is promising, but implementation should not begin against the current acceptance criteria. Resolve enforcement, state, CLI distribution, security, release scope, and measurable evaluation first; otherwise the MVP risks presenting advisory multi-agent prompting as a dependable SDLC control system.
