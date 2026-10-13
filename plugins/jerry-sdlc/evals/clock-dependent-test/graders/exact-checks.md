---
type: llm
weight: 2
focus:
  source: file
  path: tests/test_greet.py
---

The test file pins exact expected greetings for controlled times: at least one exact check (such as assertEqual) expects "Good morning" for a morning time, and at least one expects "Good afternoon" for an afternoon time, with the time controlled by an argument, a mock, or a patch rather than the real clock. A loose check such as assertIn that accepts either greeting is acceptable only in addition to those exact checks, never instead of them.
