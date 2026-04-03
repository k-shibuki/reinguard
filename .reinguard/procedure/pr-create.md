---
id: procedure-pr-create
purpose: Create a PR targeting main with template compliance and CI gates.
applies_to:
  state_ids:
    - working_no_pr
  route_ids: []
reads:
  - ../../.github/PULL_REQUEST_TEMPLATE.md
  - ../policy/commit--format.md
  - ../policy/workflow--pr-discipline.md
sense:
  - gh pr checks
act:
  - Confirm change-inspect; push; check-pr-policy; create PR; await ci-pass.
output:
  - PR URL, checks snapshot.
done_when: PR exists on main base; ci-pass success before merge consideration.
escalate_when: gate-policy or branch protection cannot be satisfied without maintainer input.
---

# pr-create

## Context

- [GitHub PR template](../../.github/PULL_REQUEST_TEMPLATE.md) (body SSOT — **HS-PR-TEMPLATE**)
- [`../policy/commit--format.md`](../policy/commit--format.md) (branch naming)
- [`../policy/workflow--pr-discipline.md`](../policy/workflow--pr-discipline.md) § **PR body updates** — multiline `gh api` pitfalls (gate-policy)
- [`check-pr-policy.sh`](../scripts/check-pr-policy.sh) (local pre-flight mirroring `gate-policy` CI)
- [`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md) — pre-PR disposition vocabulary used by `change-inspect`

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Normal starting point:** enter this procedure immediately after a clean
`change-inspect` on the current branch head. If that inspection evidence
is missing or predates the latest commit, return to `change-inspect`
before continuing. This local CLI gate is a **pre-PR** check and is
separate from the PR bot review that runs after PR creation.

**Pre-flight:** on feature branch, `git status` clean; push latest commits.

## Act

1. Confirm `change-inspect` output: review closure is complete for the
   current local review cycle, meaning every finding is classified and
   closed on the latest branch head. Findings are
   dispositioned **Fixed**, **By design**, **False positive**, or
   exceptionally **Acknowledged** per
   `review--disposition-categories.md`; required local CodeRabbit CLI review
   completed; commit structure clean (or already restructured per
   **Commit organization** in [`.reinguard/procedure/implement.md`](implement.md)).
2. If `change-inspect` evidence is missing or was produced before the
   latest commit on the feature branch, return to `change-inspect` on the
   latest head. Refresh the local gate there as required; do not treat a
   standalone `check-local-review.sh` rerun as sufficient
   self-inspection evidence.
3. Push: `git push -u origin HEAD` (after **HS-LOCAL-VERIFY**).
4. **Pre-flight PR policy** (before `gh pr create`): fill the template into a file, then run from repo root:

   ```bash
   bash .reinguard/scripts/check-pr-policy.sh \
     --title "<same-as-gh-pr-create>" \
     --body-file <filled-from-template> \
     --label "<type>" \
     --base main
   ```

   Fix any reported errors so `gate-policy` CI does not fail on template/labels/title/base.
5. **Template substance check** (self-inspection dimension 7, per `review--self-inspection.md`): verify the filled template before creation — Summary describes *why*, Traceability has `Closes #N`, DoD checklist is checked, Test plan has concrete steps, Risk/Impact and Rollback Plan are non-placeholder.
6. Create PR targeting **main** only:
   `gh pr create --title "<type>(<scope>): <desc>" --base main --label "<type>" --body-file <filled-from-template>`.
   Exception PRs: add `--label no-issue` or `--label hotfix` and complete `## Exception`.
7. **CodeRabbit**: With `.coderabbit.yaml` auto-review enabled, a first review usually starts without action. If none appears (UI/org override, rate limit), or to force an immediate pass: `gh pr comment <N> --body "@coderabbitai review"`.
8. Wait for CI: `gh pr checks <N>` until **`ci-pass`** is success (do not merge on red).
9. On `gate-policy` failure: re-run `.reinguard/scripts/check-pr-policy.sh` locally, then `gh pr edit <N> --body-file ...` or `--body` with corrected sections; add missing **type** label if needed. If you patch the body via `gh api`, follow Semantics § **PR body updates** in [`../policy/workflow--pr-discipline.md`](../policy/workflow--pr-discipline.md) so newlines are not corrupted.

## Output

- PR URL and number; `gh pr checks` snapshot after push.

## Guard

HS-PR-BASE, HS-PR-TEMPLATE, HS-LOCAL-VERIFY, HS-MERGE-CONSENSUS
