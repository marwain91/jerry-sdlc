# Efficiency control verification

Observed on 2026-10-06 in disposable Linux containers. Checks use the source, shipped Linux wrapper, and installed plugin copy. Role reports and comparison inputs are synthetic; no model-quality or real-project cost comparison was performed.

| Behavior | Observed verification |
|---|---|
| Lightweight delivery | One successful command receipt can support the owner and selected QA/verifier responsibilities; an owner without required command evidence cannot complete. |
| Scope | An unrelated edit preserves a declared candidate scope; new directory members invalidate it. Traversal and symlinks are rejected. |
| Reuse | Unchanged dependencies permit reuse with original provenance. Changed inputs, runtime identity, source evidence, missing sidecars, failed commands, and symlinks reject reuse or verification. |
| Contracts | Added specialist instruction changes invalidate frozen contracts. Old omitted-profile state keeps full role semantics. Existing full-profile and release lifecycle checks remain covered. |
| Review budgets | Two recorded rounds permit one named material-blocker extension; further review is refused. Cost overruns remain recordable and refuse new review admission. Missing counters remain unknown. |
| Waiting | Failure and hung probes return bounded outcomes. Pending jobs retain identity; command descendants are stopped, including children that hold output pipes after a pending exit. |
| Comparison | Matched synthetic results pass exploratory gates; missing measurements, unequal trials, mismatched conditions, partial token coverage, and duplicate identities are rejected. Cost, quality, and safety failures produce negative outcomes. |
| Packaging | Three targets are cross-built and reproduced byte-for-byte; wrapper checksums and CLI exit codes are checked. Native macOS execution remains unverified. |
| Installed plugin | Lightweight completion followed by completed-source check reuse and fresh reports completes through the installed wrapper. Generated run/check/dependency/ledger records conform to the shipped schemas. The installed plugin files match the source plugin directory. |

Reproduce inside the repository-authorized container:

```sh
go test ./...
go run ./cmd/jsdlc eval-workflows --fixture evals/workflow-routing-suite.json --repeats 3
go run ./cmd/jsdlc eval-comparison --fixture evals/synthetic-comparison.json
```

The routing suite covers 49 requests across 147 deterministic trials. The comparison fixture establishes harness behavior only. Schemas, catalog/fixture conformance, skill frontmatter, and formatting were also validated. Full packaging commands are in the CI workflow.

These controls do not attest dependency completeness, runtime identity, adjudication, or counters. The ledger enforces recorded review admission; it cannot stop model generation or guarantee a runtime spending cap.
