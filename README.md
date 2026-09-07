# Jerry SDLC

Jerry SDLC is a zero-init Codex plugin for risk-aware multi-agent planning, implementation, debugging, QA, code review, incident investigation, and release-readiness workflows.

The project is under phased development and must not yet be treated as a production release control. Its live team persists candidate-bound local evidence, records explicitly authorized local adjudication, audits separately authorized corrections against accepted findings and paths, and deterministically re-aggregates verdicts. Local correction scope is checked after editing, not sandbox-enforced; check execution is unattested and fully privileged, and local authorization history is rewritable by the same OS owner. Trusted assurance, portability, and evaluation gates remain open. See `PLAN.md`, `PHASE-STATUS.md`, and `PHASE-1-EVIDENCE.md`.

Contribution and security-reporting guidance is available in `CONTRIBUTING.md` and `SECURITY.md`. Jerry SDLC is licensed under the Apache License 2.0; see `LICENSE`.

The default mode requires no `jsdlc init` and adds no files to consuming repositories. A `.jsdlc.yaml` project-policy format is planned but is not parsed or enforced in the current experimental build; existing repository instructions and CI remain authoritative.

Everyday feature, bug-fix, diagnosis, PR-review, trivial-change, and incident workflows use risk-scaled Codex-native role passes. Release readiness remains the only workflow with a persisted team-evidence contract and formal verdict. See the workflow routing reference in the plugin for the exact teams and exit conditions.
