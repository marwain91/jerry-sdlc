---
max_turns: 25
timeout_seconds: 450
allowed_tools: [Read, Glob, Grep, Edit, Write, Bash, Skill, Agent]
---

The security audit flags our pinned urllib3 (CVE-2023-43804, fixed in 1.26.17). Update it and cover the security fixes.
