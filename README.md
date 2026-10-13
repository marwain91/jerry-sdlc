# Jerry SDLC

Jerry SDLC is a Codex and Claude Code plugin for proportionate software delivery. Describe the outcome; Jerry picks the smallest workflow that delivers it with credible evidence.

**Experimental.** It has not yet been shown to beat plain Codex on quality or cost. See [evaluation](docs/evaluation.md).

## Get started

### Codex

1. Register the marketplace:

   ```sh
   codex plugin marketplace add https://github.com/marwain91/jerry-sdlc.git
   ```

2. Install **Jerry SDLC** from the **Personal** marketplace using `/plugins`.
3. Start a new conversation in your project. Explicit skill selection is `$jsdlc-orchestrate`.

### Claude Code

```text
/plugin install jerry-sdlc --marketplace marwain91/jerry-sdlc
```

Or from a shell: `claude plugin marketplace add marwain91/jerry-sdlc`, then `claude plugin install jerry-sdlc@jerry-sdlc`. Explicit skill selection is `/jsdlc-orchestrate`.

### Use it

```text
Use Jerry SDLC to fix this bug and verify the regression.
```

The plugin is instructions only: no binaries, no project initialization, no generated files. Repository instructions and CI stay authoritative.

## What it does

| Request | Workflow |
|---|---|
| Typo or mechanical edit | Edit and one relevant check. |
| Bug fix or feature | One owner implements and checks; one bounded review. |
| Diagnosis or PR review | Read-only evidence or findings. |
| Incident | Authorized reversible mitigation, verified in production. |
| Release-readiness assessment | Read-only risk review; never executes a release. |

One initial review and one targeted re-review per slice. A third round needs a named blocker. Specialists are added for a named risk only. Polish becomes follow-up work. Fixes must be checked on the original failure path, and interface changes must be exercised for real.

```mermaid
flowchart LR
    A[Bounded outcome] --> B[Implement and check]
    B --> C[One review]
    C -->|Material correction| D[Fix and targeted recheck]
    D --> E[Observed outcome]
    C -->|Criteria met| E
```

## Layout

```text
plugins/jerry-sdlc/
  .codex-plugin/plugin.json          Codex plugin manifest
  .claude-plugin/plugin.json         Claude Code plugin manifest
  skills/jsdlc-orchestrate/
    SKILL.md                         routing and core rules (always loaded)
    references/workflows.md          per-workflow detail
    references/ux.md                 product design and UX checks
    references/handoffs.md           what to send delegated reviewers
.agents/plugins/marketplace.json     Codex marketplace catalog
.claude-plugin/marketplace.json      Claude Code marketplace catalog
scripts/check.sh                     structural checks run by CI
docs/evaluation.md                   how to test whether Jerry helps
```

Versions up to 0.1.x also shipped a Go CLI for evidence records, formal release verdicts, and cost ledgers. It was removed in 0.2.0; see Git history for it.

| Guide | Contents |
|---|---|
| [Evaluation](docs/evaluation.md) | Baseline-versus-Jerry comparison protocol |
| [Contributing](CONTRIBUTING.md) | Editing the instructions |
| [Security](SECURITY.md) | Trust boundaries and reporting |

Licensed under Apache License 2.0.
