# Contributing to Jerry SDLC

Jerry SDLC is evidence-driven control software. Changes are accepted on the strength of observable behavior and explicit trust boundaries, not role names or optimistic metadata.

## Development

Use Go 1.23 or the repository's current `go.mod` version. On shared development hosts, run builds and tests in a disposable container so host dependencies remain untouched:

```sh
podman run --rm --userns=keep-id \
  -e GOTELEMETRY=off -e GOCACHE=/tmp/go-cache \
  -v "$PWD:/src:z" -w /src docker.io/library/golang:1.23-alpine \
  sh -lc '/usr/local/go/bin/gofmt -w cmd/jsdlc/*.go && /usr/local/go/bin/go test ./...'
```

Keep canonical policy in `plugins/jerry-sdlc/workflows`, role contracts in `plugins/jerry-sdlc/roles`, and interchange definitions in schemas. Adapters translate mechanics; they must not duplicate or weaken canonical policy.

## Pull requests

- Explain the user-visible outcome and trust-boundary impact.
- Add behavioral tests for success, malformed input, stale identity, replay, and interruption where applicable.
- Never turn an observed or self-reported capability into attested assurance.
- Keep packs and additional workflows gated until the published Phase 3 thresholds pass on fresh evidence.
- Update `PHASE-STATUS.md` truthfully. Implementation is not graduation evidence.
- Do not include tokens, credentials, production data, or model transcripts containing secrets.
- Do not publish, tag, deploy, or create a release as part of an ordinary contribution.

Generated binaries under `plugins/jerry-sdlc/assets/bin` must be rebuilt from the reviewed source for all declared targets with `CGO_ENABLED=0`, `-buildvcs=false`, `-trimpath`, and `-ldflags='-s -w'`; `checksums.sha256` must be refreshed in the same change. Disabling VCS stamping is required so source-directory and clean-checkout builds are byte-identical. CI compares the committed binaries with those reproducible cross-builds.

## Evaluation integrity

Freeze holdout fixtures in a commit before first execution. Preserve the first result even when it fails, including the fixture digest, evaluator commit, command, timestamp, metrics, and truncated-output status. Tuned replays are regression evidence, not fresh holdouts. Historical quality datasets require human-adjudicated ground truth and may not be replaced with invented passing samples.

## Licensing gate

No license has been selected yet. Until the owner adds one explicitly, the repository is source-visible but not licensed for reuse or redistribution. Contributions must not assume a future license.
