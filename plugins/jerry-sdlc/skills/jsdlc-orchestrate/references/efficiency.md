# Cost and waiting controls

Split large requests into bounded outcomes with acceptance criteria, one review question, and a stopping point. Preserve the original objective. Checkpoints retain completed work, unmet criteria, exact candidate/job identities, evidence, and next action; they do not replace completion.

## Review and budgets

Use one initial review and one targeted correction review per slice. Further review needs a named material blocker. Optional polish becomes follow-up work. Avoid resending the entire repository or assigning several roles the same answered question.

Set provisional limits before costly work. Optional ledgers default to two reviews, 3,600 active seconds, and 500,000 uncached-input-plus-output tokens per slice. Tune them to scope. They are operating limits, not measured optimal values or spending authorization. Include worker and compaction costs when available; track cached input separately. Gross tokens are not monetary charges or quota usage.

Keep a private ledger outside the consuming repository:

```sh
slice-start --file <ledger.json> --id <neutral-id> --max-reviews 2 --max-active-seconds 3600 --max-accounted-tokens 500000
slice-record --file <ledger.json> --event <event.json>
slice-status --file <ledger.json>
```

Prefix commands with the bundled CLI path. Events follow schemas/slice-event.schema.json. IDs are unique; measurements are deltas, not repeated cumulative counters.

```json
{"id":"usage-1","kind":"measurement","evidence":"Runtime observation including workers","usage":{"inputTokens":12000,"cachedInputTokens":8000,"outputTokens":1000},"tokenCoverage":"ALL_WORKERS","activeSeconds":120,"waitSeconds":30}
```

Omit unknown measurements; status reports null/UNKNOWN. Use PARTIAL for incomplete worker coverage. Findings record CONFIRMED, REJECTED, or UNRESOLVED with severity and evidence. Record escaped defects and reopened tasks only after observation. Exclude secrets, customer data, raw prompts, and transcripts.

Recorded token/time exhaustion refuses further review admission. The normal round limit permits one extra review with a materialBlocker, then requires a checkpoint. Measurement/checkpoint events remain recordable so overruns stay visible. The ledger cannot stop model generation, enforce an external runtime cap, attest counters, or authorize a larger budget. Partial or unknown costs do not establish actual budget compliance.

At a limit, stop repeated review and save a checkpoint. Continue necessary authorized work through a bounded next slice or justified revised budget. Do not split unchanged work merely to reset counters. Never call budget exhaustion completion.

## Shared preflight

Discover the permitted runtime and canonical checks once. Before workers share an uncertain prerequisite, execute one cheap probe:

```sh
preflight --authorized --timeout 30s -- <project-authorized-probe-command>
```

Cache the observation for that slice; repeat only after relevant environment changes. Exit codes: 0 PASSED, 2 FAILED, 3 TIMED_OUT, 1 invalid input/execution error. Output digests avoid persisting raw command output. Commands use full local privileges in the caller's environment; this is not a sandbox. doctor is needed only for managed worker capability. A shared prerequisite failure should not fan out to every reviewer.

## Bounded waiting

Prefer tool completion signals. Otherwise poll the exact existing job:

```sh
wait-check --job <job-id> --authorized --timeout 60s --interval 5s --pending-exit 3 -- <status-command>
```

The command returns 0 for successful terminal state, the pending code for queued/running, and another nonzero code for failure. Each subprocess and the total poll are bounded. Result JSON preserves job identity and attempts. Exit 3 stops polling and retains the same external job for later observation. Do not restart jobs, cancel CI, or change candidates because observation timed out. Diagnose missing/failed jobs from authoritative state. Do useful independent work or yield while waiting.

## Outcome comparison

eval-comparison compares baseline, lightweight Jerry, and full Jerry on the same candidate, declared environment conditions, and fixed trial counts. It reports active/wait time, worker-inclusive cached/uncached/output costs, adjudicated findings, escaped defects, reopened work, and human effort. Read docs/evaluation.md in the source repository for the pilot protocol. Synthetic inputs establish harness behavior only; pilots cannot replace graduation evidence or authorize releases.
