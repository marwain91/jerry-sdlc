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

## Pending Phase-1 gates

- Linux managed-mode runtime-adapter persistence and automatic independent worker isolation
- macOS arm64 and x86-64 surface tests (cross-compiled binaries are present but unverified on macOS)
- IDE-extension and Codex-app matrix tests
- 100-prompt, three-repeat trigger suite
- collision/coexistence fixtures
- process interruption integration test plus schema upgrade/rollback tests (unit-level resume and incompatible-schema coverage now pass)
- published per-cell evidence table

This evidence authorizes continued Phase-1 work only. It does not establish release-readiness capability.
