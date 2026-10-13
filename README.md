# Jerry SDLC

Jerry SDLC is a Codex and Claude Code plugin for proportionate software delivery. Describe the outcome; Jerry picks the smallest workflow that delivers it with credible evidence.

**Experimental.** It has not yet been shown to beat a plain agent on quality or cost. The bundled eval suite measures that; see [evaluation](docs/evaluation.md).

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

### Upgrade

Neither tool refreshes its copy of the marketplace automatically. After a new version lands on `main`, refresh the marketplace, then update the plugin.

Codex:

```sh
codex plugin marketplace upgrade personal
codex plugin add jerry-sdlc@personal
```

Claude Code:

```sh
claude plugin marketplace update jerry-sdlc
claude plugin update jerry-sdlc@jerry-sdlc
```

Start a new conversation to load the new version. `codex plugin list` and `claude plugin list` show the installed version.

### Use it

```text
Use Jerry SDLC to fix this bug and verify the regression.
```

The plugin is instructions plus one small read-only hook in Claude Code: no binaries, no project initialization, no generated files. Repository instructions and CI stay authoritative.

## What it does

| Request | Workflow |
|---|---|
| Typo or mechanical edit | Edit and one relevant check. |
| Bug fix or feature | One owner implements and checks; one bounded review. |
| Diagnosis or PR review | Read-only evidence or findings. |
| Incident | Authorized reversible mitigation, verified in production. |
| Release-readiness assessment | Read-only risk review; never executes a release. |

One initial review and one targeted re-review per slice. A third round needs a named blocker. Specialists are added for a named risk only. Polish becomes follow-up work. Interface changes must be exercised for real.

- **Red, then green.** A bug fix starts with a test that fails for the reported reason, and the handoff shows that run and the passing run after the fix.
- **Cross-model review.** For features and risky fixes, the reviewer comes from the other model family when its CLI is installed and signed in: Claude Code asks Codex (`codex exec -s read-only`), and Codex asks Claude Code (`claude -p --permission-mode plan`). Otherwise a same-model reviewer is used, and the handoff says so.
- **Test guard (Claude Code only).** A `Stop` hook compares the session's starting commit with the working tree. If test files gained skip markers, lost assertions, or were deleted, it asks the agent once to justify the change or restore coverage. It runs `git diff` only and never runs tests. Codex plugins cannot ship hooks.

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
  hooks/hooks.json, test-guard.sh    Claude Code test guard
  evals/                             claude plugin eval suite
.agents/plugins/marketplace.json     Codex marketplace catalog
.claude-plugin/marketplace.json      Claude Code marketplace catalog
scripts/check.sh                     structural checks run by CI
scripts/test-hook.sh                 tests for the test guard
docs/evaluation.md                   how to test whether Jerry helps
```

Versions up to 0.1.x also shipped a Go CLI for evidence records, formal release verdicts, and cost ledgers. It was removed in 0.2.0; see Git history for it.

| Guide | Contents |
|---|---|
| [Evaluation](docs/evaluation.md) | Baseline-versus-Jerry comparison protocol |
| [Contributing](CONTRIBUTING.md) | Editing the instructions |
| [Security](SECURITY.md) | Trust boundaries and reporting |

Licensed under Apache License 2.0.
