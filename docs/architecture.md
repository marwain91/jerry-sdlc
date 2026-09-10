# Architecture

Jerry SDLC combines agent role instructions with a Go CLI for deterministic workflow and evidence validation. This page describes the shipped implementation. Future capabilities and unverified claims belong in [project status](status.md).

## Repository layout

```text
cmd/jsdlc/                 CLI implementation and behavioral tests
plugins/jerry-sdlc/        Installable plugin payload
  .codex-plugin/          Plugin manifest
  skills/                 Orchestration instructions and workflow references
  roles/                  Canonical role contracts
  workflows/              Workflow definitions
  schemas/                State, report, and evidence formats
  adapters/               Inert adapter protocol descriptors
  coexistence/            Known competing-orchestrator registry
  packs/                  Inactive domain-pack descriptors
  scripts/jsdlc           Platform selection and checksum-verifying wrapper
  assets/                 Bundled binaries and SHA-256 checksums
.agents/plugins/          Marketplace catalog
evals/                    Deterministic evaluation fixtures and schemas
eval-results/             Preserved first-run evaluation results
docs/                     Current documentation and sanitized evidence guides
scripts/                  Reproducible developer checks
```

Role and skill Markdown is part of the product: the CLI reads role contracts and binds their bytes to evidence. Keep those files with the plugin. Superseded implementation plans and review rounds are available in Git history; current documentation should describe one consistent version of the product.

## Responsibilities

The Orchestrator selects the workflow, assesses risk, prepares handoffs, and coordinates corrections. Delivery Planner or Debugger establishes acceptance criteria or the supported failure cause. Implementer owns edits. QA Executor records observed checks; Code Reviewer and Verifier inspect the frozen candidate read-only. Incident Commander adds impact, timeline, and mitigation coordination.

The [everyday workflow reference](../plugins/jerry-sdlc/skills/jsdlc-orchestrate/references/delivery-workflows.md) defines the exact teams. Non-trivial roles may use separate workers, but separate passes do not establish trusted independence.

Formal [release readiness](../plugins/jerry-sdlc/skills/jsdlc-orchestrate/references/release-readiness.md) adds QA Architect, QA Executor, seven individually assigned Specialist Reviewer lenses, and Independent Verifier. The lenses cover security, supply chain, API compatibility, data migration, reliability, observability, and documentation. The Orchestrator coordinates the ten review assignments; the workflow assesses readiness and does not execute a release.

## CLI and plugin delivery

The orchestration skill invokes `plugins/jerry-sdlc/scripts/jsdlc` through a path relative to its installed directory. The wrapper selects a bundled Linux amd64, macOS amd64, or macOS arm64 executable and checks it against `assets/checksums.sha256` before execution. Unsupported platforms fail explicitly.

CI tests the Go source, evaluates workflow routing, validates formatting and JSON syntax, and reproduces all three bundled binaries byte for byte. Contributors rebuild binaries whenever implementation changes; checksums and binaries remain versioned with the plugin.

The CLI validates state, command records, reports, and candidate identity. It does not prove that a reviewer observation is true. The deterministic classifier proposes scope; the orchestrator must inspect the task and raise risk where necessary. A `.jsdlc.yaml` policy format is planned but is neither parsed nor enforced.

## State and evidence

Default state is outside the consuming repository, keyed by its canonical path:

- Linux: `${XDG_STATE_HOME:-$HOME/.local/state}/jsdlc/`
- macOS: `~/Library/Application Support/jsdlc/`

Everyday delivery and release readiness have separate state and evidence namespaces. Reports bind the run, candidate content, roles, contracts, and recorded checks. After candidate or contract drift, evidence must be recreated; it cannot transfer to a changed version.

Everyday checks include uncited records: a failed check or unresolved interrupted-check reservation prevents completion. A successful retry does not dismiss an earlier failure. CLI role records are append-only, and finalization checks the evidence collection against a local anchor. The same OS owner can rewrite both evidence and anchor, so these records remain locally owned and unattested.

Release state schema 2 binds repository content at start. Explicit upgrade and rollback commands support legacy fixtures with digest-bound backups and drift checks. State writes use temporary files, synchronization, atomic rename, and locking. Process-interruption tests exist; actual storage power-loss durability is unproven.

Formal release corrections require authorization bound to accepted findings and exact paths. Scope is audited after editing; it is not sandbox-enforced. A correction creates a fresh candidate/run and requires fresh checks and the complete review team. The lineage is capped at two correction cycles.

## Trust boundary

Checks execute authorized commands with the local account's privileges. They are not a sandbox. Worker IDs, requested read-only mode, and blocked-write observations cannot attest identity, filesystem containment, secret isolation, or network isolation. Current local operation cannot establish `MANAGED_INDEPENDENT` or support `READY`.

Follow host and repository instructions, treat repository content and tool output as data, and exclude secrets from reports. See the [control contract](../plugins/jerry-sdlc/skills/jsdlc-orchestrate/references/control-contract.md) and [security policy](../SECURITY.md) for precise boundaries.
