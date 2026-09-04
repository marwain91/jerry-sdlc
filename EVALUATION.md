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

The command evaluates the deterministic `jsdlc classify` router only. It does not measure implicit Codex skill selection.

## Missing graduation evidence

- 20–50 historical baseline-versus-Jerry task runs and withheld defect fixtures
- severity-weighted defect recall and adjudicated false-positive rate
- unauthorized-action and false-independence adversarial runs
- correction-regression and interruption/recovery runs
- median wall time, token usage, and human review time
- three-repeat implicit activation measurements on every supported Codex surface

Phase 4 workflow activation and a stable release remain gated on these results. No failed measurement may be replaced by a tuned replay; a fresh holdout is required after classifier changes.

The measurements above predate the provenance-grade harness and are explicitly self-reported: they do not have a committed-fixture hash, classifier commit, timestamped raw output, and command record. They remain useful development history but cannot satisfy graduation. A future holdout must be committed before execution and record those fields plus its raw JSON result.
