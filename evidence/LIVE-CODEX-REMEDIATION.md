# Live Codex remediation observation

Date: 2026-09-05 UTC  
Candidate: `17e0820e4880dab0435f668881c57f45658431df`  
Environment: rootless `golang:1.23-alpine` project container, repository mounted read-only, disposable copied Codex credentials and disposable Jerry state

## Outcome

The current ten-assignment `team` workflow completed with ten distinct reported Codex thread IDs and persisted a candidate-, contract-, schema-, prompt-, transcript-, report-, role-, and assignment-bound receipt for every pass. The aggregate result was:

- assurance: `MANAGED_SEPARATE_PASSES`
- worker observation: `OBSERVED_DISTINCT_SUBPROCESSES`
- verdict: `INCONCLUSIVE`
- reason: `one or more workers or applicable release-risk domains are blocked`
- repository digest: `2ae2874c37d4515c75c4dfb9bbbaa3cdd571f281cd449b272c3d1706b544e7df`
- contract-set digest: `4b2f9150ce41210645da1db9cffade91ab20236bf541c83f07443748365468ac`
- team-evidence digest: `f0f7406bac26aa847d5ca933be24a7585ba63bf7963993217ecda0285546ef99`

Every assignment independently encountered the same environment limitation when attempting repository commands: nested Codex bubblewrap could not mount `devpts` inside the rootless project container (`bwrap: Can't mount devpts on /newroot/dev/pts: Permission denied`). Workers therefore returned `BLOCKED` domains with no invented checked evidence. The independent verifier reproduced the failure and did not accept earlier reports as conclusions. No worker edited the read-only candidate, accessed production, or attempted deployment, publication, tagging, or release.

This is a successful fail-closed remediation observation, not a candidate-quality assessment and not trusted independence evidence. The project execution policy forbids moving the Go runtime/test workflow onto the control host merely to bypass nested sandboxing. A model-backed repository inspection therefore remains blocked until a container-compatible trusted runtime adapter exists.

## Integration defect found during the rerun

Two initial attempts stopped before worker execution because the Codex executable was not discoverable through an absolute copied symlink. After mounting the executable at its original absolute path, Codex rejected `worker-result.schema.json`: structured output does not permit the `uniqueItems` keyword. Commit `17e0820` removed only that unsupported generation constraint; Jerry's deterministic validator continues to reject duplicate evidence IDs. A direct structured-output probe then passed, and the full team completed with the truthful `INCONCLUSIVE` result above.

The team state lived only in the disposable container and was destroyed on exit, so the path printed in the raw result is intentionally not presented as durable evidence. The stable observation is this committed summary plus the reproducible code/tests; it does not claim a retained attestation bundle.
