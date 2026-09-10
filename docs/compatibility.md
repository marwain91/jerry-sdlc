# Portability status

Jerry SDLC keeps workflow policy in canonical role, workflow, and schema files. Runtime adapters may translate execution mechanics but must not copy or weaken that policy.

| Runtime | Status | Evidence |
|---|---|---|
| Codex CLI, Linux amd64 | Experimental | Bounded live-team and packaged-lifecycle observations; write, identity, secret, and network attestation remain unavailable |
| Codex CLI, macOS arm64/amd64 | Gated | Cross-built binaries only; no native execution evidence |
| ChatGPT desktop / Codex app surfaces | Unverified | No complete Jerry activation/isolation matrix |
| Codex IDE extension | Unavailable for plugin installation | Current upstream plugin documentation excludes the IDE extension |
| Windows / Linux arm64 | Unsupported | No packaged executable |
| Claude Code | Gated descriptor | Protocol-record validation exists, but no executable adapter or behavioral validation |
| Generic runtime | Gated protocol | Canonical-policy-bound protocol records can be validated; no conforming runtime implementation exists |

Adapter protocol v2 defines a strict, bounded record tied to the exact canonical workflow, role, assignment, result-schema, and result bytes. `jsdlc validate-adapter --file <record.json> --result <worker-result.json> --assignment <assignment>` rejects missing or unknown fields, malformed worker results, byte or contract drift, invalid replay identity, and attested claims in a self-reported record. Successful validation is `EVIDENCE_ONLY`, never activates an adapter, and cannot establish `MANAGED_INDEPENDENT`. Its replay ID identifies duplicate evidence but local validation does not provide a trusted replay registry. Authentication, replay prevention, and isolation attestation require a future trusted adapter boundary.

Upstream installation-surface availability follows [OpenAI's plugin documentation](https://learn.chatgpt.com/docs/plugins#use-plugins-from-a-supported-surface). Platform availability does not validate Jerry's behavior on that surface.

The adapter catalogue reports capabilities individually. A runtime must never infer `MANAGED_INDEPENDENT` from a descriptor, process count, self-reported thread IDs, or a structurally valid protocol record.

Coexistence uses a runtime-neutral, deterministic boundary. A host supplies observed or user-declared active-orchestrator evidence matching `schemas/collision-input.schema.json`, then calls `jsdlc resolve-collision --file <evidence.json>`. Explicit selection has priority, the same active Jerry run resumes, and an unresolved competing orchestrator or different Jerry run returns `COLLISION` with `OWNER_GATE`. Resolution always reports `launchPerformed: false`.

On Codex CLI, `jsdlc inspect-codex-plugins --file <captured-json>` consumes bounded output from `codex plugin list --json`. It considers only installed and enabled entries, excludes Jerry by exact plugin ID, and classifies competitors only through the versioned `coexistence/catalog.json` registry. Unknown plugins remain unclassified; names and descriptions are deliberately not guessed. It parses but never follows or opens reported plugin source paths.

The [2026-09-07 Linux CLI observation](evidence/plugin-coexistence.md) contained Jerry SDLC and four unrelated plugins. Jerry was recognized once; the other entries remained unclassified and did not create a collision. This proves bounded inventory ingestion for that sample, not discovery of arbitrary orchestrators, implicit skill behavior, or desktop/macOS behavior. The shipped competitor registry remains empty until a real competing plugin ID and scope can be verified; synthetic exact-ID tests exercise the blocking path meanwhile.

Frontend, backend, mobile, database, infrastructure, and incident pack metadata is present but deliberately inactive. Pack and canonical lens descriptors are versioned, compatibility- and policy-digest-bound, and declare deterministic triggers, dependencies, and symmetric conflicts. `jsdlc packs --conform-set <ids>` only validates dependency closure and reports a stable merged descriptor; it always returns `activationAllowed: false` and never selects or executes a pack. All shipped catalogue, workflow, and schema references are checked as non-symlink, in-root, non-dangling files. Pack activation requires the published [evaluation requirements](evaluation.md#graduation-requirements); current results do not satisfy them.
