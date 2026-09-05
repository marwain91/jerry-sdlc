# Portability status

Jerry SDLC keeps workflow policy in canonical role, workflow, and schema files. Runtime adapters may translate execution mechanics but must not copy or weaken that policy.

| Runtime | Status | Evidence |
|---|---|---|
| Codex CLI, Linux amd64 | Experimental | Real four-role execution and write-blocking observation; write, identity, secret, and network attestation remain unavailable |
| Codex CLI, macOS arm64/amd64 | Gated | Cross-built binaries only; no native execution evidence |
| Codex IDE/app surfaces | Gated | No complete activation/isolation matrix |
| Claude Code | Gated descriptor | Protocol-record validation exists, but no executable adapter or behavioral validation |
| Generic runtime | Gated protocol | Canonical-policy-bound protocol records can be validated; no conforming runtime implementation exists |

Adapter protocol v2 defines a strict, bounded record tied to the exact canonical workflow, role, assignment, result-schema, and result bytes. `jsdlc validate-adapter --file <record.json> --result <worker-result.json> --assignment <assignment>` rejects missing or unknown fields, malformed worker results, byte or contract drift, invalid replay identity, and attested claims in a self-reported record. Successful validation is `EVIDENCE_ONLY`, never activates an adapter, and cannot establish `MANAGED_INDEPENDENT`. Its replay ID identifies duplicate evidence but local validation does not provide a trusted replay registry. Authentication, replay prevention, and isolation attestation require a future trusted adapter boundary.

The adapter catalogue reports capabilities individually. A runtime must never infer `MANAGED_INDEPENDENT` from a descriptor, process count, self-reported thread IDs, or a structurally valid protocol record.

Coexistence uses a runtime-neutral, deterministic boundary. A host supplies observed or user-declared active-orchestrator evidence matching `schemas/collision-input.schema.json`, then calls `jsdlc resolve-collision --file <evidence.json>`. Explicit selection has priority, the same active Jerry run resumes, and an unresolved competing orchestrator or different Jerry run returns `COLLISION` with `OWNER_GATE`. The command never discovers installed plugins and always reports `launchPerformed: false`; host-specific discovery remains unproven until tested on each surface.

Frontend, backend, mobile, database, infrastructure, and incident pack metadata is present but deliberately inactive. The project plan permits activation only after release/PR evaluation evidence passes; current Phase-3 results do not.
