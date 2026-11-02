---
type: regex
target:
  source: file
  path: requirements.txt
pattern: '(^|\n)urllib3\s*(==|~=|>=)\s*1\.26\.(1[7-9]|[2-9][0-9])\b'
---

The active requirement line pins or floors urllib3 at a compatible 1.26.x release with patch 17 or later. A 2.x jump is a major upgrade the user did not ask for; a commented line does not count.
