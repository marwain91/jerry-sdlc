---
max_turns: 30
timeout_seconds: 600
allowed_tools: [Read, Glob, Grep, Edit, Write, Bash, Skill, Agent]
---

Exporting customers and importing them back loses rows when a name contains a comma. Reproduce with `python3 -m app.roundtrip`: it should print `3 customers` but prints fewer. Please fix it. Tests run with `python3 -m unittest discover -s tests`.
