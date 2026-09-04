# Phase 1 feasibility evidence

Status: in progress

## Linux x86-64 / Codex CLI

- Plugin scaffold validation: PASS
- Skill validation: PASS
- Go unit tests in disposable `golang:1.23-alpine` container: PASS
- Bundled Linux binary build: PASS
- Wrapper SHA-256 verification: PASS
- `doctor`, `classify`, `start`, and `status` smoke test: PASS
- Local marketplace add and plugin install: PASS
- Fresh-session implicit trigger for “Prepare this project for release”: PASS (1 observation; formal trigger suite pending)
- Correct read-only-sandbox degradation to `ADVISORY_ONLY`: PASS
- Repository mutation during zero-init activation: none
- State resume, legal-transition, candidate-drift, incompatible-schema, lock-contention, symlink-identity, terminal rollover, and CLI/persisted-state false-READY tests: PASS; crash-interruption proof remains pending
- `MANAGED_INDEPENDENT` is fail-closed until cryptographic/runtime adapter attestation exists: PASS
- Independent reviewer in a separate read-only Codex process: observed; durable test evidence and automatic in-session delegation remain pending
- Real four-role `team` integration against commit `716f226db6ecfa7bfe9b1918123ecea1cf6f4977`: PASS for orchestration, frozen repository digest, distinct reported subprocess IDs, structured reports, and truthful `INCONCLUSIVE`; produced actionable contract-freezing, semantic-validation, candidate-binding, and diagnostic-leak findings

## Pending Phase-1 gates

- Linux diagnostic probe, role-worker receipts, and one-process four-role `team` orchestration: live-tested as `MANAGED_SEPARATE_PASSES`; remediation re-test and a trusted adapter for independent identity attestation remain pending
- macOS arm64 and x86-64 surface tests (cross-compiled binaries are present but unverified on macOS)
- IDE-extension and Codex-app matrix tests
- 100-prompt, three-repeat trigger suite
- collision/coexistence fixtures
- process interruption integration test plus schema upgrade/rollback tests (unit-level resume and incompatible-schema coverage now pass)
- published per-cell evidence table

This evidence authorizes continued Phase-1 work only. It does not establish release-readiness capability.

## Phase-3 evaluation log

- Deterministic classifier development suite: 150 labelled prompts × 3 repeats; initial 42.9% precision / 60% recall, then 100% / 100% after correction.
- First withheld adversarial suite: 100 labelled prompts × 3 repeats; 66.7% precision / 100% recall. This is a recorded failed gate, not overwritten by subsequent tuning.
- Second holdout after the first correction: 100 labelled prompts × 3 repeats; 60% precision / 90% recall. This is self-reported pre-provenance evidence, not a graduation artefact. Phase-3 trigger graduation remains failed even though the development and first adversarial suites now replay at 100% / 100%.
- Provenance-grade Holdout 3, frozen at commit `20eb53b`: 100 labelled prompts × 3 repeats; 60% precision / 30% recall; `passed: false`. Raw metrics and provenance are recorded under `eval-results/`.
- Provenance-grade Holdout 4, frozen at commit `f2c0973`: 100 labelled prompts × 3 repeats; 58.8% precision / 100% recall; `passed: false`. Phase-3 trigger graduation remains failed.
- These results cover only `jsdlc classify`; implicit Codex skill selection, baseline quality uplift, cost, latency, and historical-task evaluation remain unproven.
