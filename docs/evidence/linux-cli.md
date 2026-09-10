# Linux CLI support-matrix evidence

Scope: Codex CLI plugin payload on Linux x86-64. This is not evidence for native
macOS, Codex IDE, Codex app, storage power loss, or trusted worker isolation.

## Reproduction

From the repository root:

```sh
podman run --rm --userns=keep-id \
  -e GOTELEMETRY=off -e GOCACHE=/tmp/go-cache \
  -v "$PWD:/src:ro,z" -w /src golang:1.23-alpine \
  sh scripts/check-linux-cli.sh
```

Expected terminal marker: `LINUX_CLI_MATRIX_PASS`.

| Capability | Procedure assertion | Meaning |
|---|---|---|
| Plugin discovery | Manifest and orchestration skill are readable | Required discovery files are in the payload |
| Bundled CLI integrity | `roles` succeeds through `scripts/jsdlc` | The wrapper selected Linux amd64 and matched its SHA-256 |
| External state | `doctor` creates state below disposable `XDG_STATE_HOME` | No consuming-repository initialization is needed |
| Truthful degradation | Doctor returns `UNAVAILABLE` without Codex or `MANAGED_SEPARATE_PASSES` without independent/read-only claims when Codex is discoverable | Environment discovery never becomes independent assurance |
| Persistence/resume | Separate `start` and `status` processes return the same non-stale candidate | State survives process exit outside the repository |
| Collision | Caller-declared competitor yields `COLLISION`, `OWNER_GATE`, and no launch | Jerry does not silently start a second workflow |
| Migration/rollback | Focused migration, rollback-drift, and interruption tests pass | Legacy fixtures migrate reversibly and unsafe rollback fails |

## Observation

On 2026-09-05 UTC, the command completed in `golang:1.23-alpine` with
`LINUX_CLI_MATRIX_PASS`; the focused Go test reported `ok`. Doctor returned
`MANAGED_SEPARATE_PASSES`, with both independent-worker and read-only-isolation
claims false, after proving that its external state path was writable.

The procedure does not run `doctor --probe-independent`: subprocess IDs and a
blocked-write observation are not trusted attestation. The strongest local
live-team claim remains `MANAGED_SEPARATE_PASSES` when Codex is present.
