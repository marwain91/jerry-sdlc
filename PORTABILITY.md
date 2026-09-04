# Portability status

Jerry SDLC keeps workflow policy in canonical role, workflow, and schema files. Runtime adapters may translate execution mechanics but must not copy or weaken that policy.

| Runtime | Status | Evidence |
|---|---|---|
| Codex CLI, Linux amd64 | Experimental | Real four-role execution and write-blocking observation; write, identity, secret, and network attestation remain unavailable |
| Codex CLI, macOS arm64/amd64 | Gated | Cross-built binaries only; no native execution evidence |
| Codex IDE/app surfaces | Gated | No complete activation/isolation matrix |
| Claude Code | Gated descriptor | No executable adapter or behavioral validation |
| Generic runtime | Gated protocol placeholder | No conforming implementation |

The adapter catalogue reports capabilities individually. A runtime must never infer `MANAGED_INDEPENDENT` from a descriptor, process count, or self-reported thread IDs.

Frontend, backend, mobile, database, infrastructure, and incident pack metadata is present but deliberately inactive. The project plan permits activation only after release/PR evaluation evidence passes; current Phase-3 results do not.
