# Jerry SDLC evaluation status

Status: **not graduated**.

## Reproducible classifier evidence

Run in the project container:

```sh
go run ./cmd/jsdlc eval-triggers --fixture evals/trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/adversarial-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-2-trigger-suite.json --repeats 3
```

| Suite | Labelled prompts | Trials | First observed precision | First observed recall | Current replay |
|---|---:|---:|---:|---:|---|
| Development | 150 | 450 | 42.9% | 60% | 100% / 100% |
| Adversarial 1 | 100 | 300 | 66.7% | 100% | 100% / 100% after tuning |
| Holdout 2 | 100 | 300 | 60% | 90% | failed; self-reported pre-provenance result |
| Holdout 3 | 100 | 300 | 60% | 30% | failed; provenance-grade result at `20eb53b` |

The command evaluates the deterministic `jsdlc classify` router only. It does not measure implicit Codex skill selection.

## Missing graduation evidence

- 20–50 historical baseline-versus-Jerry task runs and withheld defect fixtures
- severity-weighted defect recall and adjudicated false-positive rate
- unauthorized-action and false-independence adversarial runs
- correction-regression and interruption/recovery runs
- median wall time, token usage, and human review time
- three-repeat implicit activation measurements on every supported Codex surface

Phase 4 workflow activation and a stable release remain gated on these results. No failed measurement may be replaced by a tuned replay; a fresh holdout is required after classifier changes.

The first three measurements predate the provenance-grade harness and are explicitly self-reported: they do not have a committed-fixture hash, classifier commit, timestamped raw output, and command record. They remain useful development history but cannot satisfy graduation. A future holdout must be committed before execution and record those fields plus its raw JSON result.

Holdout 3 is the first provenance-grade result. The fixture was committed before execution, its SHA-256 is `3c7b92d8a93822d65c0b88bebbba1b23ddd84f597eb63420dc3e5e71d7a784bb`, and [the recorded result](eval-results/holdout-3-20eb53b.json) binds the command, timestamp, classifier commit, metrics, and failure-list truncation. It failed, so classifier graduation remains blocked.
