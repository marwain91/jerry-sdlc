# Lightweight real-task observations

Use this optional template after ordinary Jerry-assisted work to learn what helped,
what failed, and what to improve. It requires no extra model runs, project setup,
or automatic collection. The formal repeated benchmark remains deferred.

## Keep the record private by default

Copy the template to an approved private location outside the consuming repository
and its frozen candidate. Do not commit raw task records to this public repository.
Recording is optional and must follow the project's data-retention rules. Nothing
here enables telemetry, uploads, or a new CLI command.

Use neutral task IDs. Omit customer/project names, private repository URLs, local
paths, hostnames/IPs, credentials, personal data, proprietary code, and raw prompts,
transcripts, or command output. Keep sensitive evidence only in an approved private
store; a digest does not make sensitive source material safe to publish. Record a
sanitized observation or `withheld` when necessary. Before any public contribution,
obtain the data owner's approval and review the exact diff manually; a secret scan
alone is not sufficient.

## One short record per task

Use `unknown` or `not measured` for unavailable facts, never an estimated zero.
Copy outcomes from observed evidence. Do not infer a passed test from a role report.

```text
Task ID / date:
Task category / selected workflow / risk:
Plugin version / candidate identity (private reference if sensitive):
Sanitized intended outcome / acceptance criteria:
Roles actually run / separate passes or self-review:
Observed checks (sanitized purpose, exit status, evidence reference):
Findings (confirmed / rejected / unresolved, with supporting evidence):
Corrections / recheck results:
Engineering outcome / unmet criteria:
CLI outcome (exact value, or not run) / assurance limitations:
Elapsed time / source (or not measured):
Model tokens / source (or not measured):
Human review time / source (or not measured):
Later regression or escaped defect / follow-up date (or not yet observed):
One useful workflow improvement:
```

For elapsed time, state whether pauses and human wait time are included. Record
token totals only when the runtime exposes a task-scoped total, including workers;
otherwise mark them unavailable or explicitly partial. Do not derive weekly quota
usage from token estimates. A reviewer finding is not a confirmed defect until it
has supporting evidence; unresolved disagreements remain unresolved.

Where feasible, keep a consistent task-selection rule (for example, all voluntarily
recorded bug fixes in a chosen week), including failures and abandoned work. Record
the rule and omissions with any summary. Preserve the original observation when
adding corrections or later regressions; lack of a reported defect is not evidence
that none escaped.

## What these records can establish

They can identify practical workflow friction, observed failures, useful review
findings, and candidates for a future privacy-approved evaluation corpus. They are
uncontrolled observations, not a baseline-versus-Jerry comparison. Do not claim
quality uplift, representative defect recall, false-positive rates, budget ratios,
or graduation from anecdotes or selectively recorded successes.

This template is not input to `delivery-record` or `eval-quality`, and does not
replace candidate-bound CLI evidence. Everyday `COMPLETE` has no release-readiness
effect; separate agents do not by themselves establish attested independence.
No observation authorizes publishing, deployment, or a release.

Before resuming the formal benchmark, obtain owner approval for the corpus and
budget, establish adjudicated ground truth, and run the repeated baseline/Jerry
protocol and [published evaluation thresholds](../evaluation.md#graduation-requirements). Do not retrofit an invented baseline onto these records.
