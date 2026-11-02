# Contributing to Jerry SDLC

The product is the Markdown under `plugins/jerry-sdlc/skills/`. Codex and Claude Code load the same skill; each has its own manifest. Every sentence there costs model attention on every task, so changes are judged by whether they change agent behavior for the better.

## Guidelines

- **State each rule once.** `SKILL.md` holds routing, the workflows, and the rules every task needs. References hold detail that only some tasks need (UX, delegation). Do not split out a reference that nearly every task reads; that only adds a tool call. Do not repeat a rule across files.
- **Keep `SKILL.md` small.** CI fails it above 950 words. Move detail to a reference instead of raising the limit.
- **Write instructions, not caveats.** Say what the agent should do. Trust-boundary disclaimers belong in `SECURITY.md`, not in the skill.
- **Back behavior changes with evidence.** Describe the observed failure that motivates a rule, or run the comparison in [docs/evaluation.md](docs/evaluation.md). A new rule should come with an eval case that it moves. A rule nobody has seen fail is a candidate for deletion.
- **No hooks without real-use evidence.** The 0.3.x test guard was removed because it never caught a real problem and replaced the user's final handoff with justifications. A new hook must show true positives on real sessions first.
- When the plugin changes, bump `version` in both `plugins/jerry-sdlc/.codex-plugin/plugin.json` and `plugins/jerry-sdlc/.claude-plugin/plugin.json` so Codex and Claude Code refresh their caches. Keep the base versions equal.
- Do not include tokens, credentials, production data, or transcripts containing secrets.

## Checks

`scripts/check.sh` validates JSON, the skill front matter, the word limit, relative links, eval case structure, and shell syntax. On shared hosts, run it in a disposable container:

```sh
podman run --rm -v "$PWD:/src:ro,z" -w /src docker.io/library/python:3.12-alpine \
  sh scripts/check.sh
```

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
