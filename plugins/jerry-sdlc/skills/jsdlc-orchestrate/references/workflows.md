# Workflows

Roles are responsibilities, not agents. One owner can plan, implement, test, and verify. Delegate only when a separate pass adds real coverage.

## Bug fix

1. Establish the failure on the reported path: entry point, starting state, action sequence, expected and actual result. Include navigation, reload, timing, or persistence when they matter. If you cannot reproduce it, say which part stays unverified.
2. Treat any suggested cause, including the user's, as a hypothesis until evidence supports it.
3. Inspect the layer where the failure shows. Correct network data or successful storage does not prove correct rendering or client state.
4. Fix, then check the original path and the behavior that must stay the same. Cover affected consumers (for example web and mobile clients of a shared API) or name the ones left untested.
5. If the fix does not work, compare the remaining symptom with the original and pick a check that tells competing causes apart before you try again.

## Feature

1. Inspect before proposing architecture. Infer routine choices from context; ask only about consequential missing decisions.
2. Check that external integrations actually support what you plan before you promise compatibility.
3. Implement the bounded outcome, including the loading, empty, error, and recovery states it needs.
4. One reviewer covers correctness and, when relevant, focused UX.

A planning-only request ends with the plan, acceptance criteria, and open decisions. Do not implement.

## Diagnosis

Read-only unless the user authorizes a fix. Return the supported cause, the evidence, competing hypotheses still open, and the next distinguishing check. Get a second opinion only when the conclusion is consequential and uncertain.

## PR or design review

Read-only, within the requested scope. Return findings ranked by impact, each with location, evidence, and a concrete fix. Run extra checks only to close a specific coverage gap. Do not invent findings to justify the review.

## Incident

Keep an impact statement, timeline, hypotheses, decisions, and next safe action. Prefer reversible mitigation and preserve evidence. Production, destructive, credential, and external-communication actions still need their normal authorization. A passing repository check does not prove production recovery; observe the live effect.

## Release-readiness assessment

Only when the user asks for one explicitly. This is a read-only assessment of an exact candidate (commit SHA or labeled working tree). It does not tag, publish, or deploy.

Review only the lenses that apply, and say which ones you skipped and why:

- security (authn/authz, secrets, injection, dependency CVEs)
- supply chain (pinned dependencies and actions, build reproducibility)
- API and data compatibility (public interfaces, migrations, rollback)
- reliability and observability (failure modes, timeouts, logs and alerts for new paths)
- documentation and release notes

Run the project's full required checks on the candidate. Report a verdict: **ready**, **not ready** (with blocking findings), or **inconclusive** (with the missing evidence). Any change to the candidate voids the assessment.

If a release is separately authorized, follow the project's own release procedure and verify target versions, artifacts, metadata, and published state for each target.

## Tracked tasks

When the user authorizes task updates, reconcile the saved acceptance criteria with what was delivered, then read back the checklist and status. Do not invent a deployment step for an implementation-only task. A requested deployment stays open until you verify it.
