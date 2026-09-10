# Codex CLI plugin coexistence evidence

Observed on 2026-09-07 on Linux amd64 with the public plugin inventory captured by `codex plugin list --json` and immediately inspected by the rebuilt Jerry CLI. Local source paths and remote connector IDs are intentionally omitted.

The sample contained `jerry-sdlc@personal` and four unrelated installed, enabled plugins. Incidental third-party plugin names are omitted; this is a sanitized observation rather than a reusable inventory fixture.

The result reported `jerryPluginDetected: true`, no registry-confirmed competitors, and the four non-Jerry IDs as unclassified. The exact captured inventory digest was `12b2c351eddcbf2fea9698b7b2e08b007128f8fba2983c391e7c91d0bd886016`; the shipped empty-registry digest was `7effb3a1d75275eb393659a48d65c7190137cff49123853553be703a7abb83b5`.

This proves bounded ingestion and benign coexistence for the listed inventory on this host. It does not prove that an unclassified plugin is harmless, discover arbitrary broad orchestrators, or exercise an actual competing orchestrator. The blocking branch is synthetic until a real competitor ID and scope are verified.
