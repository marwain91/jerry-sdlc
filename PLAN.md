# Jerry SDLC implementation plan

Status: revision 6. The control plane, release-readiness vertical slice, evaluation harness, and inert portability contracts are implemented experimentally and the source repository is public under Apache-2.0. Graduation, workflow/pack activation, and a stable release remain gated by the exit criteria below.

## 1. Product outcome

Jerry SDLC (`jsdlc`) is a public Codex plugin that a user activates for a project. After activation, the user continues to work normally. Requests such as “plan this feature”, “implement this”, “review this”, and “prepare a release” are expected to trigger the appropriate SDLC workflow and team of narrow specialist agents.

Implicit skill selection is model-driven, so automatic triggering is a measured convenience rather than an enforcement claim. The plugin always exposes an explicit `$jsdlc`/`jsdlc` fallback. The default experience requires no generated project files, dependency installation, or `jsdlc init`. Optional project policy may be added for customization or hard enforcement.

## 2. Product principles

1. Activation is the onboarding: install/enable the plugin, pass its automatic smoke check, and begin working.
2. The user states the outcome; the orchestrator selects the workflow and roles.
3. Roles are permission and evidence contracts, not personas.
4. Use deterministic tools for facts and agents for judgment.
5. Reviewer independence must be real or truthfully labelled self-review.
6. Workflow depth is proportional to change risk.
7. Release authorization and other material human decisions remain human gates.
8. The plugin must complement existing repository instructions, tests, and CI rather than replace them.

## 3. User experience

After the plugin is active, normal prompts drive the workflow:

- “Investigate this bug” triggers the diagnosis workflow.
- “Implement this feature” triggers planning, implementation, testing, and relevant review.
- “Review this PR” triggers risk classification and specialist review lenses.
- “Prepare this for release” triggers the release-readiness workflow, including a QA team rather than a single generic QA pass.

At the start, Codex briefly reports the selected workflow, risk level, assurance mode, and team. It proceeds automatically until it reaches a genuine owner decision or release authorization. If implicit triggering does not occur, `$jsdlc prepare this for release` is the explicit, reliable entry point.

The activation smoke check verifies plugin discovery, skill availability, bundled validator compatibility, subagent capability, and writable state storage. It returns one deterministic activation outcome from the table below; it never silently degrades assurance.

| Outcome | Meaning | Permitted behavior |
|---|---|---|
| `MANAGED_INDEPENDENT` | CLI, state, distinct workers, and read-only isolation work | Full workflow and independent assurance |
| `MANAGED_SEPARATE_PASSES` | CLI/state work but independent isolation does not | Workflow continues with visible `SELF_REVIEW`; never satisfies an independence gate |
| `ADVISORY_ONLY` | Skills work but CLI/state validation does not | Advice and role prompts only; no `READY` verdict |
| `UNAVAILABLE` | Plugin or skill cannot run reliably | Stop with installation diagnostics |

## 4. Architecture

### 4.1 Plugin as the delivery mechanism

The public repository contains a Codex plugin with:

- an orchestration skill with broad SDLC trigger language;
- canonical runtime-neutral role contracts;
- Codex agent adapters;
- workflow definitions;
- evidence schemas and deterministic validators;
- optional domain skills and workflow packs.

Installing and activating the plugin makes these capabilities available to the project. No repository mutation is required for the default mode.

### 4.2 Orchestration skill as the entry point

The `jsdlc-orchestrate` skill triggers for software planning, implementation, QA, review, release, deployment preparation, maintenance, and incident work. It:

1. reads the user request and repository context;
2. classifies the work type and risk;
3. selects a workflow and required roles;
4. delegates narrow assignments with immutable inputs;
5. collects and adjudicates findings;
6. requests corrections and changed-area re-review;
7. verifies the result against the repository and tools;
8. stops at required human gates.

The skill does not claim guaranteed implicit activation. Trigger reliability is tested as a product metric. Explicit `$jsdlc` invocation is the supported fallback and the required entry point for enforced workflows when the host runtime cannot guarantee implicit selection.

Trigger precedence is deterministic within Jerry:

1. an explicit user invocation of a named skill/workflow wins;
2. an already active Jerry run resumes unless the user replaces it;
3. the most specific matching Jerry workflow wins over the general orchestrator;
4. specialist skills are inputs to the selected workflow, not competing orchestrators;
5. if another installed plugin claims orchestration of the same request, Jerry reports the collision and does not start a second workflow unless explicitly selected.

### 4.3 `jsdlc` CLI delivery and responsibility

The CLI supports the plugin; users normally do not drive it. It is implemented in Go and shipped inside the plugin as versioned standalone executables plus source for each supported OS/architecture. It is invoked by the orchestration skill through a path relative to the installed `SKILL.md`, so no `PATH` modification is required. The plugin manifest pins the CLI version and contains SHA-256 digests; `jsdlc doctor` verifies the selected binary before execution. Marketplace source/version trust remains the installation trust root; Jerry does not invent a separate signing authority in v1. Plugin and CLI schema versions negotiate on startup and fail closed on incompatibility.

Upgrade and rollback are plugin-version operations: each plugin release contains exactly one compatible CLI/schema set. Before migrating state, the new CLI writes an atomic backup and migration journal; failure leaves the old state untouched. Downgrades may read state only when the recorded schema is compatible, otherwise the user rolls back the plugin and restores the matching automatic backup. In-progress runs remain pinned to their originating workflow/schema version unless explicitly migrated and revalidated.

The initial support matrix is:

| Codex surface | Linux x86-64 | macOS arm64 | macOS x86-64 | Windows |
|---|---:|---:|---:|---:|
| CLI | Phase-1 managed target | Phase-1 managed target | Phase-1 managed target | Unsupported in v1 |
| IDE extension | Phase-1 managed target | Phase-1 managed target | Phase-1 managed target | Unsupported in v1 |
| Codex app | Phase-1 discovery/advisory target | Phase-1 managed target | Phase-1 managed target | Unsupported in v1 |

“Target” is not a compatibility claim until the Phase-1 matrix test passes. A failed managed target becomes `ADVISORY_ONLY`, visibly, and cannot emit readiness. Other platforms return `UNAVAILABLE` or advisory mode; they are not silently treated as supported.

A managed-target cell passes only if installation, explicit invocation, CLI digest verification, state persistence, interruption/resume, worker isolation, and expected `MANAGED_*` outcome all pass. An advisory-target cell passes only if installation, explicit invocation, collision reporting, and the expected `ADVISORY_ONLY` outcome pass. A failed cell blocks advertising that cell as supported; removing it from v1 requires a recorded product decision rather than silently weakening its expected outcome.

Its responsibility is deterministic state, policy, and evidence validation, not model judgment. Risk classification has two parts: deterministic triggers from changed paths/manifests/diff features, and an agent-proposed semantic classification. Policy applies the higher risk and records the reason.

Initial commands:

- `jsdlc classify`: classify work and select a workflow.
- `jsdlc roles`: resolve required and optional roles.
- `jsdlc check`: run structural and evidence checks.
- `jsdlc status`: summarize the active workflow run.
- `jsdlc verify`: validate final evidence and unresolved blockers.
- `jsdlc doctor`: run activation and capability diagnostics.

### 4.4 State and recovery

Every non-trivial workflow receives a run ID and immutable candidate revision. State is persisted outside the consuming repository under `${XDG_STATE_HOME:-$HOME/.local/state}/jsdlc/` on Linux and `~/Library/Application Support/jsdlc/` on macOS, keyed by a hash of the canonical repository path. No project file is required. `jsdlc doctor` tests this location under the active sandbox; inability to write selects `ADVISORY_ONLY`. A project can explicitly opt into committed `.jsdlc/runs/` evidence for CI or auditability.

The versioned state machine records:

- workflow, schema, role, policy, plugin, and CLI versions;
- repository identity, candidate commit/tree hash, and dirty-diff digest;
- current state and legal next transitions;
- role assignments and runtime capability handshake;
- input/output hashes and finding dispositions;
- command evidence, failures, retries, and human decisions;
- cancellation, interruption, and terminal disposition.

On resume, any candidate or input drift invalidates dependent evidence. Tool failure becomes `INCONCLUSIVE` or `BLOCKED`, never an inferred pass. Correction cycles default to two before owner escalation. Flaky or conflicting evidence cannot produce release readiness.

### 4.5 Optional project configuration

After v1 graduation, projects may opt into a future `.jsdlc.yaml` to define:

- authoritative build and test commands;
- protected paths and risk rules;
- supported platforms and environments;
- mandatory specialist roles;
- human approval gates;
- advisory, managed, or enforced operation.

`.jsdlc.yaml` is not parsed or enforced by the current implementation. Until that explicitly versioned policy format exists, Jerry SDLC infers repository capabilities conservatively from existing instructions, manifests, CI, and test configuration; repository instructions remain authoritative. Documentation must not instruct users to create the file yet.

### 4.6 Policy precedence and security

Jerry SDLC follows host instruction precedence; it never attempts to override system, developer, user, repository, sandbox, or infrastructure controls. Project content, diffs, issue text, logs, test output, and generated files are treated as untrusted data, not new instructions. Conflicting repository guidance is surfaced to the user.

The default plugin has no production credentials and performs no release. Roles receive least-privilege tool surfaces where the runtime supports them. Read-only review requires a separate read-only worker/context; otherwise the run is labelled `SEPARATE_PASSES` and cannot satisfy a policy requiring independence. External writes, deployments, publishing, destructive actions, paid services, and secret access retain the host's authorization requirements.

Any release authorization is recorded against the exact candidate commit/tree hash and, when present, build-artifact digest. Any mutation expires it. Command evidence records the command, working directory, exit status, timestamp, candidate digest, and captured result without storing secrets.

## 5. Canonical role catalogue and MVP workers

### Orchestration and planning

- Workflow Orchestrator
- Intent and Requirements Analyst
- Solution Architect
- Implementation Planner
- Plan Reviewer

### Implementation

- Software Engineer
- QA Engineer
- Corrector
- Technical Writer

### Quality and review

- QA Architect
- Correctness Reviewer
- Test Adequacy Reviewer
- Security Reviewer
- API Compatibility Reviewer
- UX and Accessibility Reviewer
- Independent Verifier

### Release

- Release Manager
- Production Verifier

Each contract defines its trigger, required immutable inputs, expected structured output, allowed tools, prohibited actions, escalation conditions, and independence requirements.

The catalogue above defines responsibilities, not a requirement for eighteen simultaneous workers. The MVP uses five operational workers and loads specialist lenses as needed:

- Orchestrator/Release Manager;
- QA Architect;
- QA Executor;
- Specialist Reviewer, configured with exactly one lens;
- Independent Verifier.

Implementation and correction are included only when the user also authorizes fixes. Additional standalone roles graduate only when evaluations show a distinct evidence or permission boundary.

## 6. Core workflows for v1

### 6.1 Trivial change

Implementer → relevant deterministic checks → concise verification.

### 6.2 Bug fix

Diagnosis → implementation plan → implementer → QA Engineer → Correctness Reviewer → accepted corrections → Independent Verifier.

### 6.3 Feature

Requirements → architecture/plan → Plan Reviewer → QA Architect test strategy → implementer → QA Engineer → triggered specialist reviewers → corrections → Independent Verifier.

### 6.4 Pull-request review

Diff risk classification → relevant independent review lenses → deduplicated, evidence-backed findings ranked by severity. Read-only by default.

### 6.5 Release readiness

Release Manager establishes exact candidate and change scope → QA Architect creates risk-based strategy → QA Engineers execute functional, integration, regression, and exploratory checks as applicable → specialist security/API/UX/migration reviewers trigger from the diff → findings are adjudicated and corrected → fresh Independent Verifier reruns critical evidence → release-readiness report → explicit owner authorization before release.

This is the flagship workflow and must demonstrate that “prepare this for release” automatically assembles a competent QA and review team.

The MVP release workflow is explicitly scoped to **release-readiness assessment**, not deployment or release execution. It must inspect and cover, or declare `NOT_APPLICABLE` with evidence, at least:

- tests, build, lint, and static validation;
- changed behavior and regression scope;
- authentication/security and dependency/supply-chain risk;
- API and data/schema compatibility;
- migration, rollback, and partial-failure risk;
- reliability, observability, and operational readiness;
- documentation and release notes;
- immutable candidate/build identity.

If the repository lacks enough evidence to assess one applicable area, the verdict is `INCONCLUSIVE` or `NOT_READY`; generic “ready” is forbidden.

MVP ownership and minimum evidence:

| Domain | Responsible lens | Minimum evidence or escalation |
|---|---|---|
| Functional/regression | QA Architect + QA Executor | Acceptance-to-test map and observed command/manual results |
| Security/supply chain | Specialist Reviewer: security | Changed trust boundaries, dependency delta, available scanners; high-risk ambiguity blocks |
| API compatibility | Specialist Reviewer: compatibility | Interface/schema diff and known consumers; unknown public consumers makes result inconclusive |
| Data/migration | Specialist Reviewer: migration | Forward path, compatibility window, rollback/forward-fix, existing-data test; destructive/unknown migration blocks |
| Reliability/operations | Specialist Reviewer: reliability | Failure/retry/idempotency analysis plus relevant tests and telemetry expectations |
| Observability | Specialist Reviewer: operations | New failure modes mapped to logs/metrics/alerts or evidenced `NOT_APPLICABLE` |
| Documentation/release notes | Specialist Reviewer: documentation | User/operator impact mapped to current docs and release notes |
| Candidate/build identity | Independent Verifier | Exact tree/commit and artifact digests; drift blocks |

Each lens has a schema with required evidence and permitted `NOT_APPLICABLE` reasons. One Specialist Reviewer worker may run multiple isolated lenses sequentially, but the report does not present those lenses as independent workers. High-risk security, destructive migration, or production-topology uncertainty requires `OWNER_GATE` in v1.

## 7. Risk-based role selection

The orchestrator starts with the smallest sufficient workflow and adds roles from observable triggers:

- authentication, authorization, secrets, or untrusted input → Security Reviewer;
- public API or schema change → API Compatibility Reviewer and Technical Writer;
- database schema/data change → mandatory MVP migration lens; a later pack may promote it to a dedicated worker;
- user-interface change → UX and Accessibility Reviewer plus exploratory QA;
- concurrency, retry, queue, or distributed state → mandatory MVP reliability lens; a later pack may promote it to a dedicated worker;
- release request → QA Architect, QA Engineer, Release Manager, and Independent Verifier always required.

Unknown high-impact conditions escalate rather than silently selecting a weak workflow.

## 8. Independence and assurance

Every workflow records one of:

- `FULL_INDEPENDENT`: distinct eligible workers implement and perform final review.
- `SEPARATE_PASSES`: the same worker performs isolated passes from frozen inputs; all output is labelled self-review.
- `OWNER_GATE`: required independence or evidence is unavailable, so the workflow stops at the gate.

Multiple role names executed by one context must never be presented as an independent team.

At workflow start, a capability handshake records whether the runtime can create distinct workers, enforce read-only tools, isolate worktrees/snapshots, and preserve state. A deterministic degradation matrix selects the strongest truthful mode. Policies may require `FULL_INDEPENDENT`; if unavailable, the only legal transition is `OWNER_GATE`.

## 9. Findings and correction protocol

Reviewers are read-only and return structured findings containing:

- stable finding ID;
- exact file/location and evidence;
- severity and confidence;
- violated requirement or observed failure;
- narrow recommended outcome.

The orchestrator deduplicates and adjudicates findings. Only accepted findings enter the Corrector’s allowlist. After correction, affected checks and review lenses run again. Reviewers do not opportunistically edit code.

Reviewer disagreement is preserved, not averaged into consensus. Blocking disagreement becomes an owner decision unless deterministic evidence resolves it. Duplicate findings retain provenance. Every run has explicit `READY`, `NOT_READY`, `INCONCLUSIVE`, `CANCELLED`, or `BLOCKED` terminal status.

## 10. Repository layout

The tree below is conceptual. In the current repository, distributable content
is rooted at `plugins/jerry-sdlc/`, Go CLI source is at `cmd/jsdlc/`, validators
are CLI subcommands, and captured evidence lives under `evidence/` and
`eval-results/`.

```text
jerry-sdlc/
├── .codex-plugin/plugin.json
├── skills/jsdlc-orchestrate/SKILL.md
├── roles/                         # canonical contracts
├── adapters/codex/                # Codex agent definitions
├── workflows/                     # risk-aware workflow definitions
├── schemas/                       # role, workflow, finding, evidence
├── packages/cli/                  # jsdlc executable
├── checks/                        # deterministic validators
├── evals/                         # historical and adversarial fixtures
├── examples/                      # complete example runs
└── docs/
```

## 11. Implementation phases

### Phase 1: control-plane spike and feasibility gates

- Scaffold the public Codex plugin and marketplace metadata.
- Prove zero-init activation, bundled CLI discovery, and user-state persistence on supported platforms.
- Execute and publish every cell in the declared support matrix, including binary digest verification, schema mismatch, upgrade, and rollback tests.
- Measure implicit skill triggering and document the explicit `$jsdlc` fallback.
- Define the state machine, candidate identity, capability handshake, degradation matrix, policy precedence, and security model.
- Prove separate read-only reviewer execution and truthful fallback where isolation is unavailable.

Exit criterion: all feasibility gates pass on the declared Codex surfaces. Later experimental code may be developed behind fail-closed gates, but no release-readiness or support claim graduates before this foundation works.

Phase-1 evidence uses at least four representative repositories: a small library, web application, service/API, and multi-component project. Each explicit invocation and lifecycle test runs once per matrix cell; implicit trigger classification uses at least 100 labelled prompts with three repeated trials per supported surface.

### Phase 2: narrow release-readiness vertical slice

- Implement the release-readiness workflow using the five MVP workers.
- Cover the minimum release risk domains listed above.
- Implement `jsdlc doctor`, `classify`, `roles`, `check`, `status`, and `verify`.
- Add structured findings, adjudication, hash/staleness checks, and assurance labelling.
- Bind readiness and authorization evidence to immutable candidate/artifact digests.
- Define cancellation, inconclusive outcomes, flaky tests, disagreement, budgets, and correction limits.

Exit criterion: “prepare this for release” selects the QA team, produces a defensible `READY`/`NOT_READY`/`INCONCLUSIVE` verdict, and cannot release or deploy anything.

### Phase 3: evaluation, hardening, and PR review

- Build fixtures from 20–50 real historical tasks.
- Compare baseline single-agent runs with Jerry workflows.
- Add withheld synthetic/adversarial fixtures to reduce historical-task leakage.
- Measure trigger precision/recall, severity-weighted defect recall, false-positive rate, evidence fabrication, unauthorized actions, correction regressions, human review time, cost, and latency over repeated trials.
- Add adversarial tests for false independence, stale inputs, invented test results, ignored failures, and unauthorized release.
- Add coexistence tests for repository skills and competing broad orchestration plugins.
- Gate changes to roles and skills on eval results.
- Evaluate the experimental read-only PR-review workflow before treating it as graduated or enabling release enforcement from its output.

Initial graduation thresholds:

- explicit `$jsdlc` activation: 100% on supported surfaces;
- implicit release-intent trigger recall: at least 95%, with at least 98% precision on the labelled trigger suite;
- zero unauthorized external writes or releases in adversarial tests;
- zero false independent-assurance claims;
- zero `READY` verdicts with known blocking fixture defects;
- at least 25% improvement in severity-weighted escaped-defect detection over the generic single-agent baseline;
- false-positive findings no worse than 15% after adjudication;
- median wall-clock time no more than 3× the generic single-agent baseline when independent work can run concurrently;
- median model-token usage no more than 5× baseline and no individual run above a configurable hard budget;
- if quality thresholds require exceeding either budget, v1 remains blocked until the owner explicitly changes the published product budget.

Exit criterion: thresholds pass across repeated runs and the results, including failures and cost, are published.

### Phase 4: portability and ecosystem

- Add experimental Codex-native trivial-change, diagnosis, bug-fix, feature, PR-review, and incident workflows for practical iteration. Do not represent them as graduated or as having the release workflow's persisted evidence contract until evaluation is satisfactory.
- Add Claude Code and generic adapters without duplicating canonical policy.
- Add frontend, backend, mobile, database, infrastructure, and incident packs.
- Add optional CI enforcement and persistent audit artefacts.
- Prepare stable marketplace releases and contribution standards.

## 12. Acceptance criteria by phase

### Phase 1

- Activation, CLI trust, persistence, isolation, upgrade/rollback, collision, and support-matrix gates above pass.
- Explicit invocation succeeds in every supported cell; implicit triggering is measured but does not block the feasibility spike unless explicit invocation is also unreliable.

### Phases 2–3 / v1 graduation

1. Installing and activating the plugin is sufficient for default use; no `init` step is required.
2. Explicit `$jsdlc` invocation works on every supported Codex surface, and ordinary-language trigger behavior meets the published precision/recall threshold.
3. “Prepare this for release” invokes QA Architect, QA Engineer, relevant specialist reviewers, Release Manager, and an independent final verifier when runtime capability permits.
4. Workflow depth changes according to measured risk and repository scope.
5. In `MANAGED_INDEPENDENT`, reviewers are technically read-only and corrections require accepted finding IDs. If read-only isolation is unavailable, the workflow degrades visibly or stops according to policy and cannot claim this guarantee.
6. The system never labels same-worker passes independent.
7. Test commands and claimed results are verified from tool output.
8. Release execution remains blocked until the user gives explicit authorization.
9. The plugin respects existing `AGENTS.md`, repository commands, CI, and infrastructure constraints.
10. Eval fixtures demonstrate better defect detection than a single generic review without unacceptable false-positive or latency growth.
11. Activation diagnostics prove CLI compatibility, writable state, and available isolation before claiming managed assurance.
12. Every run has a versioned state record and survives interruption without accepting stale evidence.
13. Release readiness covers applicable migration, rollback, reliability, observability, dependency, security, compatibility, and documentation risks or returns `INCONCLUSIVE`.
14. Authorization is bound to an immutable candidate and expires on change.
15. Every declared support-matrix cell has a published activation, isolation, persistence, upgrade, and failure-mode result.
16. Trigger collisions never launch two orchestration workflows silently.

## 13. Initial non-goals

- A hosted orchestration service.
- Automatic work triggered from GitHub issues without an active agent session.
- Mandatory files added to every consuming repository.
- Dozens of decorative personas.
- Hard enforcement without an explicit project opt-in.
- Supporting every agent runtime in the first release.
- Deploying, publishing, tagging, or otherwise executing a release in the MVP.

## 14. Principal risks

- Plugin skill triggering may be advisory rather than guaranteed; mitigate with precise trigger descriptions, evals, and optional project enforcement.
- Multiple agents may still have correlated blind spots; require narrow lenses and independent evidence, not vote counting.
- Full workflows may be too expensive for routine changes; use risk-based selection and publish cost/latency measurements.
- Generic inference may choose incorrect project commands; prefer existing repository instructions and fail closed when execution is unsafe or unclear.
- Role proliferation may create review noise; require every role to own a distinct decision or evidence source.
- Untrusted repository content may attempt to redirect the workflow; treat repository data as untrusted and enforce host instruction precedence and tool boundaries.

## 15. First development milestone

Build a vertical slice around one sentence: **“Prepare this project for release.”**

The slice must install as a Codex plugin, pass `jsdlc doctor`, support explicit invocation with measured implicit triggering, inspect a representative repository, assemble the release QA team, execute permitted checks, return structured findings, perform only separately authorized corrections, independently verify the exact immutable candidate, and produce `READY`, `NOT_READY`, or `INCONCLUSIVE`. It must not deploy, publish, tag, or release.
