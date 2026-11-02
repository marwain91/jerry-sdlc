---
type: llm
focus:
  source: file
  path: src/pricing.py
---

`discount` subtracts the percentage once, so discount(200, 10) returns 180 and discount(200, 0) returns 200.
