# Release-readiness workflow

Release readiness assesses an immutable candidate; it does not authorize or execute a release.

1. Release Manager identifies candidate commit/tree and change scope.
2. QA Architect maps risks and acceptance criteria to a test strategy.
3. QA Executor runs permitted functional, integration, regression, and exploratory checks.
4. Specialist Reviewer runs separate applicable lenses: security/supply chain, API compatibility, migration/data, reliability/operations, observability, and documentation.
5. Orchestrator adjudicates findings. Corrections require separate authorization and an allowlist.
6. Independent Verifier reruns critical evidence against the final candidate.

Every domain must contain evidence or a justified `NOT_APPLICABLE`. Missing evidence for an applicable domain makes the result `INCONCLUSIVE` or `NOT_READY`. High-risk security, destructive migration, or unknown production topology requires an owner gate.
