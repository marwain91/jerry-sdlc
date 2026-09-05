# Jerry SDLC evaluation status

Status: **not graduated**.

## Synthetic evaluator harness evidence

Two 20-task, three-trial-per-arm fixtures exercise the quality evaluator without
claiming observations from models, historical projects, or users. The passing
fixture uses adjudicated synthetic findings and in-budget timing/token values;
the failing fixture injects unauthorized action, false-independence, false-READY,
evidence-fabrication, correction-regression, wall-time, token-ratio, and hard
token-budget failures.

The captured results are `eval-results/synthetic-quality-pass.json` and
`eval-results/synthetic-quality-fail.json`. The first reports every gate true;
the second reports all injected safety and budget gates false. These prove only
that the deterministic metric harness accepts and rejects its bounded inputs as
specified. They do not contribute to Phase-3 quality graduation, defect uplift,
or real cost/latency evidence.

## Reproducible classifier evidence

Run in the project container:

```sh
go run ./cmd/jsdlc eval-triggers --fixture evals/trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/adversarial-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-2-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-3-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-4-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-5-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-6-trigger-suite.json --repeats 3
go run ./cmd/jsdlc eval-triggers --fixture evals/holdout-7-trigger-suite.json --repeats 3
```

| Suite | Labelled prompts | Trials | First observed precision | First observed recall | Current replay |
|---|---:|---:|---:|---:|---|
| Development | 150 | 450 | 42.9% | 60% | 100% / 100% |
| Adversarial 1 | 100 | 300 | 66.7% | 100% | 100% / 100% after tuning |
| Holdout 2 | 100 | 300 | 60% | 90% | 100% / 100% after tuning; first result remains failed |
| Holdout 3 | 100 | 300 | 60% | 30% | 100% / 100% after tuning; first result remains failed |
| Holdout 4 | 100 | 300 | 58.8% | 100% | 100% / 100% after tuning; first result remains failed |
| Holdout 5 | 100 | 300 | 0% | 0% | 100% / 100% after tuning; first result remains failed |
| Holdout 6 | 100 | 300 | 87.5% | 70% | 100% / 100% after tuning; first result remains failed |
| Holdout 7 | 100 | 300 | 100% | 50% | 100% / 100% after tuning; first result remains failed |

The command evaluates the deterministic `jsdlc classify` router only. It does not measure implicit Codex skill selection.

## Missing graduation evidence

- 20–50 historical baseline-versus-Jerry task runs and withheld defect fixtures
- severity-weighted defect recall and adjudicated false-positive rate
- repeated model/workflow observations for unauthorized actions and false-independence (the synthetic harness gates are validated)
- repeated model/workflow correction-regression observations and interruption/recovery runs (the synthetic regression gate is validated)
- median wall time, token usage, and human review time
- three-repeat implicit activation measurements on every supported Codex surface

Phase 4 workflow activation and a stable release remain gated on these results. No failed measurement may be replaced by a tuned replay; a fresh holdout is required after classifier changes.

Both evaluators now include the raw fixture SHA-256, evaluation timestamp, and evaluation-output schema version in their JSON. Trigger results also include the total failure count and whether the displayed failure list was truncated. These fields make a captured result self-identifying, but the committed fixture and evaluator commit still have to be recorded externally; the CLI cannot prove that a fixture was frozen before its first execution.

`jsdlc eval-quality --fixture <adjudicated-suite.json>` validates a 20–50 task dataset and computes severity-weighted recall uplift, finding precision, median latency/token ratios, unsafe events, evidence fabrications, correction regressions, and the combined graduation result. [The schema](evals/quality-suite.schema.json) defines the bounded interchange format. Array bounds are structural; exact equality to the suite's `trialsPerArm` is an enforced CLI semantic constraint. The harness is present; no historical result dataset has been supplied or claimed.

Quality output includes the exact threshold values and a boolean result for every gate. Relative recall uplift is deliberately undefined when baseline recall is zero: the output reports the absolute uplift but fails `baselineRecallPositive` and the relative-uplift gate rather than inventing a percentage from a zero denominator. Such a corpus cannot establish the required comparative improvement.

The first three measurements predate the provenance-grade harness and are explicitly self-reported: they do not have a committed-fixture hash, classifier commit, timestamped raw output, and command record. They remain useful development history but cannot satisfy graduation. A future holdout must be committed before execution and record those fields plus its raw JSON result.

Holdout 3 is the first provenance-grade result. The fixture was committed before execution, its SHA-256 is `3c7b92d8a93822d65c0b88bebbba1b23ddd84f597eb63420dc3e5e71d7a784bb`, and [the recorded result](eval-results/holdout-3-20eb53b.json) binds the command, timestamp, classifier commit, metrics, and failure-list truncation. It failed, so classifier graduation remains blocked.

Holdout 4 was independently frozen at commit `f2c0973` with fixture SHA-256 `a5669d17317c4bf6daf08eb1cbecf1d2231460f0c3d745f3d20690a78d540be9`. [Its result](eval-results/holdout-4-f2c0973.json) also failed. Replaying either suite after tuning is regression evidence only, never a replacement for its first observation.

Holdout 5 was frozen at commit `02d68eb` before its first execution, with fixture SHA-256 `e21ff75dcf93a3cec2a90de1ff2500d0385200d168cd9df03e30fa034e648926`. [Its first result](eval-results/holdout-5-02d68eb.json) had 150 false negatives and no positive predictions (0% recall; precision reported as 0), so it failed. It demonstrates that the exact-phrase router does not generalize to a fresh set of ordinary release-request paraphrases. This result is preserved and may not become graduation evidence after tuning.

Holdout 6 was independently authored and frozen at commit `1a5590d` before its first execution, with fixture SHA-256 `54573ca366e7fbc87dd9ddbfcfb1719582e7d8dcba72552f7f00b9f74230e4aa`. [Its first result](eval-results/holdout-6-1a5590d.json) had 87.5% precision and 70% recall, so it failed both thresholds. The misses cover risk evaluation, launch-blocker discovery, and store-submission readiness; publishing internal documentation caused false positives. This result is preserved and cannot serve as graduation evidence after tuning.

Holdout 7 was independently authored at `af11452`, but its first preflight was rejected before classification because it used the wrong JSON field names. Commit `279d378` mechanically corrected only the schema mapping without changing prompt wording, then froze fixture SHA-256 `7eef5eb60d93a2c2a9d1893c2b36d90f49048144fb2604143b0c17e1707a82ac`. [Its first valid result](eval-results/holdout-7-279d378.json) had 100% precision and 50% recall, so it failed. Both the rejected preflight and valid failure are preserved; neither is graduation evidence.
