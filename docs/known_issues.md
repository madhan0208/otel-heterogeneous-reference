# Known Issue: Build Validation Branch Policy Not Triggering on PR Creation

**Status:** Unresolved, documented, deprioritized
**Date investigated:** 2026-06-21
**Environment:** Azure DevOps (personal/free-tier organization, created same day)

## Summary

A `main` branch protection policy was configured correctly (Build Validation,
Required, Automatic trigger, pipeline `Otel`, path filter `/`), but the
validation build did not fire on PR creation or update across three separate
test PRs. The branch policy correctly *blocked direct pushes* to `main`
(confirmed via `TF402455` error), proving the policy is active — but the
build-triggering half of it never executed at PR time.

## What was ruled out, in order

1. **YAML syntax errors** — file verified correct, no indentation/structure issues.
2. **`pr:` trigger block in YAML** — added, tested, confirmed via Azure DevOps's
   own UI message that PR triggers are not supported for Azure Repos Git at
   all ("Pull request triggers are not supported for Azure Repos. Please use
   build validation policies."). Removed afterward as dead configuration.
3. **Branch Policy panel settings** — every field manually re-verified twice:
   Enabled: On, Build pipeline: Otel, Path filter: /, Trigger: Automatic,
   Policy requirement: Required.
4. **Trigger-type conflict** (`trigger: main` competing with Build Validation) —
   tested by setting `trigger: none` to isolate Build Validation entirely.
   No change in behavior.
5. **Organization dormancy** (orgs sleep 5 min after last sign-out, per
   Microsoft Learn troubleshooting docs) — ruled out by pushing a fresh
   commit while actively signed in and watching the Pipelines run list in
   real time. No new run appeared.

## Evidence

Across all 3 test PRs (`test-branch-policy`, `test-branch-policy-v2`/v3,
`remove-pr-trigger`), the Pipelines run history shows **zero runs timestamped
at PR creation/update time**. The only runs present are tagged "Individual CI
for [user]" and are timestamped at *merge* time — i.e., the existing
`trigger: - main` CI trigger firing normally after merge, completely
independent of any PR-time validation.

## Why this was deprioritized rather than fully resolved

This is a personal, same-day-created, free-tier Azure DevOps organization —
not a production environment. The underlying CI/CD mechanics (compliance
testing, authenticated Azure login, Docker build/push) are fully proven and
working independently of this specific feature. Spending further time on a
single platform quirk, isolated to a throwaway learning org, had diminishing
return relative to continuing the broader SRE/DevOps learning plan.

## What this demonstrates regardless of the unresolved bug

- Correct branch policy configuration per Microsoft's documented best practice
- Systematic, hypothesis-driven debugging: each theory was tested in
  isolation, with evidence gathered before moving to the next hypothesis,
  rather than guessing repeatedly
- Awareness of platform-specific behavior differences (Azure Repos vs.
  GitHub/Bitbucket PR trigger handling)

## Possible next steps, if revisited

- Recreate the Build Validation policy from scratch (delete, re-add) rather
  than editing the existing one
- Test on a different (e.g. company-managed or longer-lived) Azure DevOps
  organization to rule out free-tier/new-org-specific behavior
- Check Azure DevOps service health status page for the relevant date, in
  case of a regional outage affecting build validation specifically
- Post on Microsoft Developer Community / Stack Overflow with this exact
  elimination list, since the issue may be a known, currently-unfixed bug
