# Release-readiness workflow

Release readiness assesses an immutable candidate; it does not authorize or execute a release.

1. Release Manager identifies candidate commit/tree and change scope.
2. QA Architect maps risks and acceptance criteria to a test strategy.
3. The orchestrator captures permitted deterministic commands with `jsdlc check`; QA Executor evaluates those records and performs bounded exploratory assessment.
4. Separate Specialist Reviewer assignments run the stable, candidate-bound lenses `specialist-security`, `specialist-supply-chain`, `specialist-api-compatibility`, `specialist-data-migration`, `specialist-reliability`, `specialist-observability`, and `specialist-documentation`. Each receipt and persisted report carries its assignment ID; missing or duplicated assignments fail closed.
5. Orchestrator proposes finding and applicability dispositions. Record only explicitly authorized decisions with `jsdlc adjudicate`, bound to the exact team-evidence digest. Corrections require a second explicit authorization containing accepted finding IDs and exact allowed paths. Completion creates a fresh candidate/run and is capped at two cycles.
6. Independent Verifier reruns critical evidence against the final candidate and cites the resulting check IDs. `jsdlc verify` then reproduces the persisted verdict.

Every `PASS` domain must cite successful candidate-bound checked evidence. `NOT_APPLICABLE` is accepted only through an explicit adjudication with rationale and relevant successful evidence IDs. Missing evidence for an applicable domain makes the result `INCONCLUSIVE` or `NOT_READY`. High-risk security, destructive migration, or unknown production topology requires an owner gate.
