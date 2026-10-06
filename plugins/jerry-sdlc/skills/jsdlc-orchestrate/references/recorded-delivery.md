# Optional everyday records

Use durable local records when requested or useful. They are not required for ordinary delivery. Prefix commands below with the bundled CLI path from SKILL.md and use the repository-authorized environment. Keep reports and manifests outside the candidate. Evidence is LOCAL_UNATTESTED.

## Profile and scope

New delivery-start records default to lightweight:

| Workflow | Roles |
|---|---|
| feature, bug-fix | implementer, code-reviewer |
| trivial-change | implementer |
| bug-diagnosis | debugger |
| pr-review | code-reviewer |
| incident | incident-commander, debugger, verifier |

The owner covers applicable planning, diagnosis, and QA. Experience focused/full records the assessment without mandatory extra roles: the owner supplies the brief; the reviewer covers experience criteria. Add roles with `--add-roles qa-executor,ux-reviewer --role-reason '<specific risk or coverage gap>'`. Pass the same selection to roles when inspecting the contract.

`--profile full` retains the larger role contract, experience additions, and extra verifier execution. Old state without a profile keeps full semantics. Generic HIGH-risk persistence remains unavailable except for the existing incident repository-evidence contract; necessary native specialist work can continue.

Candidates cover the whole repository by default. Lightweight records can use `--inputs <paths.json>`: a nonempty array such as `["src/queue","tests/queue","package-lock.json","vitest.config.ts"]`. Directories bind membership, contents, and modes. Paths cannot escape or traverse symlinks. Include affected consumers and configuration dependencies. Results explicitly report CALLER_DECLARED_INPUTS, not whole-repository or release coverage. Prefer isolated worktrees when dependency coverage is uncertain.

## Record and finalize

Inspect delivery-status before starting. Resume only a matching, non-stale ACTIVE run. Cancel stale active runs with their original candidate. Do not ignore status errors.

1. Freeze implementation, then delivery-start with repository, candidate, workflow, and selected profile/experience. Read-only work freezes at intake.
2. Capture authorized checks with `delivery-check --repo <repo> --candidate <label> --id <id> --authorized -- <argv...>`.
3. Record selected roles with delivery-record, a report following schemas/delivery-report-input.schema.json, and SELF_REVIEW or SEPARATE_PASS. Reports are append-only. A lightweight implementation workflow's owner report cites successful checks. Clean QA, UX, and verifier reports also cite checks; lightweight roles can share adequate evidence.
4. Run delivery-verify. Missing reports leave the run ACTIVE. All checks, including uncited failures, participate. Do not remove failures to turn a run green.

COMPLETE, ISSUES, INCOMPLETE, and BLOCKED apply only to the candidate and declared scope. Their releaseReadinessEffect is NONE. Finalized evidence changes block verification. The same OS owner can replace records and their anchor; no attestation is established.

## Correct and reuse

Edits require a fresh candidate record and fresh role reports. Reuse only deterministic checks whose complete dependencies and immutable runtime identity were captured during original execution:

```sh
delivery-check --repo <repo> --candidate <old> --id unit --inputs <check-inputs.json> --environment '<container-image-digest/toolchain-identity>' --authorized -- <argv...>
delivery-cancel --repo <repo> --candidate <old>
delivery-start --repo <repo> --candidate <new> --workflow feature
delivery-reuse-check --repo <repo> --candidate <new> --id unit-reused --source-run <old-run-id> --source-check unit --environment '<same-runtime-identity>'
```

Reuse executes no command. It retains the original receipt, binds original digests to the target, and labels reuse. Changed dependencies, directory membership, command identity, runtime identity, or original evidence reject reuse. Failed, timed-out, changing, incomplete, or already-reused originals cannot be reused. Missing dependency records block verification.

Include implementation, tests, transitive dependencies, lockfiles, build/test configuration, and fixtures. Dependency completeness and runtime identity are caller-declared. Live-production, network-dependent, external-state, or incomplete-dependency checks must execute again. Never transplant reviewer conclusions to changed candidates.

Interrupted executions retain reservations. Recover only with the original identity/authorization and a stopped command; recovery refuses live or indeterminate processes. Observation timeout does not authorize restarting a check.
