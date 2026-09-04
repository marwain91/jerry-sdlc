# Release-readiness workflow

Release readiness assesses an immutable candidate; it does not authorize or execute a release.

1. Release Manager identifies candidate commit/tree and change scope.
2. QA Architect maps risks and acceptance criteria to a test strategy.
3. The orchestrator captures permitted deterministic commands with `jsdlc check`; QA Executor evaluates those records and performs bounded exploratory assessment.
4. Specialist Reviewer runs separate applicable lenses: security/supply chain, API compatibility, migration/data, reliability/operations, observability, and documentation.
5. Orchestrator proposes finding and applicability dispositions. Record only explicitly authorized decisions with `jsdlc adjudicate`, bound to the exact team-evidence digest. Corrections require separate authorization and an accepted-finding allowlist.
6. Independent Verifier reruns critical evidence against the final candidate and cites the resulting check IDs. `jsdlc verify` then reproduces the persisted verdict.

Every `PASS` domain must cite successful candidate-bound checked evidence. `NOT_APPLICABLE` is accepted only through an explicit adjudication with rationale and relevant successful evidence IDs. Missing evidence for an applicable domain makes the result `INCONCLUSIVE` or `NOT_READY`. High-risk security, destructive migration, or unknown production topology requires an owner gate.
