# Release-readiness workflow

Release readiness assesses an immutable candidate; it does not authorize or execute a release.

Read [control-contract.md](control-contract.md) before this workflow. Commands below use the bundled `../../scripts/jsdlc` relative to the skill directory and must run in the execution environment permitted by repository instructions.

Derive a stable candidate label (exact commit SHA for a clean checkout; an explicit working-tree label otherwise). Inspect `status --repo <repo> --candidate <candidate>` first. Resume only a matching non-terminal run. If no state exists, use `start --repo <repo> --candidate <candidate> --workflow release-readiness --assurance <doctor-outcome>`. For a terminal run, `start` archives it automatically. Never replace a mismatched active run.

## Execution and verification

- Before the team run, an already-authorized deterministic command may be captured with `../../scripts/jsdlc check --repo <repo> --candidate <candidate> --id <stable-id> --domains <claimed-domains> --authorized -- <argv...>`. This is a fully privileged local command runner, not a sandbox or permission boundary: do not invoke it merely because the workflow wants evidence. Its records are visibly `LOCAL_UNATTESTED` and cannot satisfy readiness until an attested adapter and policy-defined check specs exist.

- If a check process is interrupted, do not delete state files manually. After confirming the exact command should no longer be running, use `recover-check --repo <repo> --candidate <candidate> --id <stable-id> --authorized`; it refuses while the owner or contained command process group is live and never overwrites committed evidence. If interruption occurred before the process-group identity became durable, recovery deliberately refuses and the safe path is a fresh run.

- Launch the complete review team with `../../scripts/jsdlc team --repo <repo> --candidate <candidate> --objective <objective>`. Do not manually substitute a single generic QA pass. PASS domain claims cite relevant check IDs, but locally self-labelled domains do not establish coverage. Reports and receipts are persisted outside the repository and bound to the candidate, checks, and frozen contracts.

- Use `worker` only for an explicitly bounded extra pass. Its receipt has `assuranceEffect: EVIDENCE_ONLY` and cannot independently establish assurance.

- The orchestrator proposes adjudication, but explicit local decisions are recorded only with `adjudicate --repo <repo> --candidate <candidate> --file <decision.json> --authorized`. Bind the file to the exact returned team-evidence digest. Accepted findings remain blocking; rejected findings require rationale and successful checked evidence. A `NOT_APPLICABLE` decision additionally requires successful check IDs relevant to that domain. Adjudication never upgrades assurance.

- A corrector receives only accepted finding IDs and bounded targets. Before editing, record the separate authorization with `authorize-correction --repo <repo> --candidate <candidate> --file <correction.json> --authorized`; its input binds the team digest, accepted finding IDs, and exact paths (use a trailing `/` only for an authorized subtree). Do not treat adjudication alone as permission to edit.

- Local mode does not sandbox the corrector: it rejects repositories containing symlinks and audits scope only after editing. Do not claim it prevented a same-user process from making and hiding external changes. After editing, run `finish-correction --repo <repo> --candidate <old> --new-candidate <new> --authorization <digest> --authorized`. It rejects visible out-of-scope or empty repository changes, caps the lineage at two cycles, and creates a fresh baselined run. Re-run affected checks and the complete team; the prior evidence cannot verify the new candidate.

Run `../../scripts/jsdlc verify --repo <repo> --candidate <candidate>` before reporting the terminal status. It reproduces the latest persisted verdict against the current candidate, checked evidence, reports, receipts, and contracts. `READY` additionally requires `MANAGED_INDEPENDENT`; clean separate passes remain `INCONCLUSIVE`. `NOT_APPLICABLE` remains `INCONCLUSIVE` until adjudication exists. Any finding produces `NOT_READY`; missing, failed, stale, or blocked evidence produces `INCONCLUSIVE` or `BLOCKED`. A readiness verdict never authorizes deployment, publishing, tagging, or release.

1. Release Manager identifies candidate commit/tree and change scope.
2. QA Architect maps risks and acceptance criteria to a test strategy.
3. The orchestrator captures permitted deterministic commands with `jsdlc check`; QA Executor evaluates those records and performs bounded exploratory assessment.
4. Separate Specialist Reviewer assignments run the stable, candidate-bound lenses `specialist-security`, `specialist-supply-chain`, `specialist-api-compatibility`, `specialist-data-migration`, `specialist-reliability`, `specialist-observability`, and `specialist-documentation`. Each receipt and persisted report carries its assignment ID; missing or duplicated assignments fail closed.
5. Orchestrator proposes finding and applicability dispositions. Record only explicitly authorized decisions with `jsdlc adjudicate`, bound to the exact team-evidence digest. Corrections require a second explicit authorization containing accepted finding IDs and exact allowed paths. Completion creates a fresh candidate/run and is capped at two cycles.
6. Independent Verifier reruns critical evidence against the final candidate and cites the resulting check IDs. `jsdlc verify` then reproduces the persisted verdict.

Every `PASS` domain must cite successful candidate-bound checked evidence. `NOT_APPLICABLE` is accepted only through an explicit adjudication with rationale and relevant successful evidence IDs. Missing evidence for an applicable domain makes the result `INCONCLUSIVE` or `NOT_READY`. High-risk security, destructive migration, or unknown production topology requires an owner gate.
