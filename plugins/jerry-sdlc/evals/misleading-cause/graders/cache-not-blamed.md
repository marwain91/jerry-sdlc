---
type: regex
weight: 2
flags: i
target:
  source: file
  path: src/cache.py
pattern: 'ttl|expir|invalidat|time\.|datetime'
match: not_contains
---
