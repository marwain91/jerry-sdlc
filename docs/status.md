# Project status

Jerry SDLC is an **experimental public project**. Available implementation and passing deterministic checks do not establish production release authority, model-quality improvement, or verified support on every platform.

## Capabilities and remaining work

| Area | Implemented and observed | Required before broader claims |
|---|---|---|
| Plugin and control plane | Bundled CLI, external state, content binding, locks, transitions, reversible state migration, and frozen contracts; Linux container tests and interruption fixtures | Native platform evidence, complete activation/coexistence testing, actual storage power-loss evidence, and trusted identity/isolation |
| Everyday delivery | Six workflows with role reports, command evidence, frozen-candidate verification, duplicate/drift rejection, and separate release state; packaged Linux lifecycle tests | Real model-backed review-quality evidence and trusted execution evidence; generic HIGH-risk delivery persistence remains unavailable except for the incident repository-evidence contract |
| Release readiness | Ten review assignments, local checked records, persisted reports, adjudication, bounded correction authorization, two-cycle cap, and deterministic verdict reproduction | Enforced correction containment, trusted authorization history, sandboxed/attested policy-defined checks, live defect fixtures, and independent assurance before `READY` |
| Evaluation | Workflow routing regression suite, original failed trigger holdouts, quality evaluator, and synthetic pass/fail harness fixtures | Historical tasks, repeated real model comparisons, implicit skill-selection measurements, and published quality/cost results |
| Portability and ecosystem | Three packaged binary targets, adapter/pack protocol validation, collision handling, and sanitized Linux inventory observations | Native macOS validation, supported-surface activation evidence, a verified real competing orchestrator, executable non-Codex adapters, and evaluated pack activation |

Default use requires no project initialization. The proposed `.jsdlc.yaml` project policy is not implemented. Domain packs and Claude/generic runtime integrations remain inactive. See [compatibility](compatibility.md) for the support matrix.

## Evidence index

These are bounded, dated observations or reproducible fixtures, not blanket support or assurance claims.

| Evidence | Scope and limitation |
|---|---|
| [Linux CLI matrix](evidence/linux-cli.md) | Discovery, checksum wrapper, external state, resume, collision refusal, and migration fixtures; no native macOS or independent isolation claim |
| [Packaged everyday lifecycle](evidence/everyday-lifecycle.md) | All six workflow lifecycles and real command interruption/recovery; role reports are synthetic |
| [Release lifecycle fixture](evidence/release-lifecycle.md) | Known finding → authorized correction → fresh review; fake adapter must never yield `READY` |
| [Live remediation observation](evidence/live-remediation.md) | Ten assignments returned truthful `INCONCLUSIVE` under a nested-container sandbox limitation |
| [Plugin coexistence observation](evidence/plugin-coexistence.md) | Sanitized inventory ingestion with unrelated plugins; no real competing orchestrator was exercised |
| [Evaluation results](evaluation.md) | Original failed holdouts, evaluator semantics, reproducible commands, and missing graduation evidence |

Earlier feasibility work observed marketplace installation, plugin/skill validation, zero-init activation without repository mutation, and one implicit release-intent activation. A four-role run against `716f226db6ecfa7bfe9b1918123ecea1cf6f4977` produced actionable contract-freezing, validation, candidate-binding, and diagnostic-leak findings while reporting `INCONCLUSIVE`. The linked ten-assignment rerun records the later environment limitation. These historical observations do not satisfy a repeated activation matrix or establish trusted worker independence; retained summaries are not attestation bundles.

## Development priorities

1. Collect privacy-safe historical task data and complete the repeated baseline-versus-Jerry benchmark. Its current status is deferred; [the published thresholds](evaluation.md#graduation-requirements) remain unchanged.
2. Establish supported-surface evidence and verified coexistence with a real competing orchestrator. Native macOS execution and desktop/IDE availability must be evaluated separately from binary cross-compilation.
3. Provide a trusted adapter that can authenticate workers and enforce write, secret, and network boundaries. Local records and protocol validators cannot substitute for it.
4. Complete evaluation before graduating workflows, enabling packs, or claiming stable support. Stable-release authorization is a separate owner decision.

Optional [private task observations](evidence/real-task-capture.md) can identify friction and candidates for an evaluation corpus. They add no telemetry or automatic data collection and do not replace controlled evaluation.
