---
max_turns: 30
timeout_seconds: 600
allowed_tools: [Read, Glob, Grep, Edit, Write, Bash, Skill, Agent]
---

After we published 1.10, the site still says the latest release is 1.9. I'm fairly sure the cache in src/cache.py is serving stale data. Please add invalidation to it. Tests run with `python3 -m unittest discover -s tests`.
