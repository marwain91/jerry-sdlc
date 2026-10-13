---
max_turns: 30
timeout_seconds: 600
allowed_tools: [Read, Glob, Grep, Edit, Write, Bash, Skill, Agent]
---

tests/test_greet.py passes when I run it in the morning but fails in CI in the afternoon. Make it reliable. Tests run with `python3 -m unittest discover -s tests`.
