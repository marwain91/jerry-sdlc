# Fake-adapter release-readiness lifecycle

This deterministic fixture exercises the real local Jerry SDLC control plane
without paid model calls, network access, deployment, publication, or writes
outside disposable test directories.

## Reproduction

From the repository root:

```sh
podman run --rm --userns=keep-id \
  -e HOME=/tmp/jsdlc-fixture-home \
  -e GOTELEMETRY=off -e GOCACHE=/tmp/go-cache \
  -v "$PWD:/src:ro,z" -w /src golang:1.23-alpine \
  sh evidence/run-fake-release-lifecycle.sh
```

Expected terminal marker: `FAKE_RELEASE_LIFECYCLE_PASS`.

## Scenarios and assertions

1. A disposable candidate contains a known `BROKEN` marker. A fake read-only
   Codex adapter returns a concrete high-severity `QA-KNOWN-DEFECT` finding.
   The complete ten-assignment team run must return `NOT_READY`.
2. The fixture explicitly accepts that exact finding, separately authorizes a
   correction limited to `defect.txt`, changes only that path, and finishes the
   correction into a fresh candidate/run.
3. Every assignment is rerun. The corrected fake adapter no longer reports the
   defect, while specialist lenses and the verifier truthfully report missing
   attested checks. Both the team result and reproduced `verify` result must be
   `INCONCLUSIVE` under `MANAGED_SEPARATE_PASSES`.

The fixture asserts the exact outcome sequence and fails if any stage returns
`READY`. The fake adapter provides deterministic protocol exercise only; it is
not evidence of model quality, trusted identity, isolation, or release safety.

