# Jerry SDLC phase status

This matrix distinguishes implementation from graduation evidence. A phase is not complete merely because its code exists.

| Phase | Implemented now | Verified now | Still required |
|---|---|---|---|
| 1 — control plane | Zero-init plugin, signed-by-checksum bundled CLI, external state, candidate content binding, locks, transitions, frozen worker contracts, truthful degradation | Linux container tests, plugin/skill validation, marketplace install, one real four-role run | Remediation live rerun; crash interruption and upgrade/rollback; coexistence; implicit activation on declared surfaces; native macOS and IDE/app cells; trusted isolation/identity adapter |
| 2 — release readiness | Four-role release team, structured findings/domains, evidence-gated verdict aggregation, no release/deploy capability | Unit and fake-adapter integration tests; independent false-READY review | Adjudication and correction protocol; `check` evidence ingestion; justified N/A handling; live clean/finding/blocking verdict fixtures |
| 3 — evaluation/hardening | Strict trigger evaluator and adjudicated 20–50 task quality-metric engine | Container tests; two provenance-grade trigger holdouts correctly failed | Fresh passing holdout; implicit skill-selection evaluation; real historical and synthetic task results; defect uplift, FP, time, token, human-review, correction, and adversarial thresholds |
| 4 — portability/ecosystem | Canonical adapter descriptor protocol and inactive specialist-pack catalogue | Strict bounded parsers, schemas, tests, independent review | Phase-3 graduation before pack/workflow activation; executable Claude/generic adapters; native platform evidence; PR-review/trivial/bug/feature workflows; CI enforcement and contribution/release preparation |

Current overall status: **not graduated and not releasable**. No repository release, tag, deployment, external publication, or production change has been performed.

External evidence limits on this VPS are recorded rather than waived: native macOS execution and complete Codex IDE/app behavior cannot be proven here; historical-task quality evaluation needs an adjudicated corpus and repeated model runs; trusted worker identity/secret/network attestation is not exposed by the current CLI boundary.
