# Evaluation

Status: **unproven.** Nobody has yet shown that Jerry beats plain Codex. This is the most important open question for the project; new features should wait for its answer.

## Protocol

Compare two arms on the same tasks: **baseline** (plain Codex, no plugin) and **Jerry** (plugin installed and invoked).

1. **Pick 10–20 real tasks** by a fixed rule decided in advance (for example, the next N closed issues across two or three repositories). Keep failed and abandoned tasks. Include bug fixes, features, and at least a few UI changes.
2. **Freeze each task:** starting commit, task text, acceptance criteria, and known defects or a hidden regression test where possible.
3. **Hold everything else fixed:** model, settings, tools, environment. Only the plugin varies.
4. **Run each arm at least 3 times per task**, in fresh sessions.
5. **Record per run:** acceptance criteria met (yes/partial/no), defects found later or by a hidden test, wall-clock time, total tokens including subagents, review rounds, and any unsafe action (unrequested deploy, destructive command, false claim of a passing check).
6. **Judge blind** where possible: score diffs without knowing which arm produced them.

## Decision rule

Jerry earns its place only if it meets the acceptance criteria at least as often as baseline, with fewer escaped defects, and without large cost overhead (a reasonable starting bar: median tokens and time at most 1.5× baseline). Zero unsafe actions in either arm is a hard requirement.

If Jerry does not win, shrink it further. Rules that do not change outcomes are cost.

Keep raw results private unless the data owner approves publication. Publish aggregate results, including failures.
