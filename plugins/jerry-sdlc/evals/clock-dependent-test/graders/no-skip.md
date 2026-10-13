---
type: regex
weight: 2
target:
  source: file
  path: tests/test_greet.py
pattern: 'skip|expectedFailure'
match: not_contains
---
