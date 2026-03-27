# pr-create

## Context

- `.github/PULL_REQUEST_TEMPLATE.md` (body SSOT — **HS-PR-TEMPLATE**)
- `.reinguard/policy/commit--format.md` (branch naming; Cursor: `commit-format.mdc`)
- `.reinguard/policy/workflow--pr-discipline.md` § **PR body updates** — multiline `gh api` pitfalls (gate-policy)
- `.reinguard/scripts/check-pr-policy.sh` (local pre-flight mirroring `gate-policy` CI)

**Already in context** (do not re-list): `reinguard-bridge.mdc` (HS-*, catalogs), `workflow-policy.mdc` (exceptions, command separation).

**Pre-requisite:** `change-inspect` completed with no Blocking findings.

**Pre-flight:** on feature branch, `git status` clean; push latest commits.

## Act

1. Confirm `change-inspect` output: no Blocking findings, commit structure clean (or restructured per **Commit organization** in `.cursor/commands/implement.md`).
2. Push: `git push -u origin HEAD` (after **HS-LOCAL-VERIFY**).
3. **Pre-flight PR policy** (before `gh pr create`): fill the template into a file, then run from repo root:

   ```bash
   bash .reinguard/scripts/check-pr-policy.sh \
     --title "<same-as-gh-pr-create>" \
     --body-file <filled-from-template> \
     --label "<type>" \
     --base main
   ```

   Fix any reported errors so `gate-policy` CI does not fail on template/labels/title/base.
4. **Template substance check** (self-inspection dimension 6, per `review--self-inspection.md`): verify the filled template before creation — Summary describes *why*, Traceability has `Closes #N`, DoD checklist is checked, Test plan has concrete steps, Risk/Impact and Rollback Plan are non-placeholder.
5. Create PR targeting **main** only:
   `gh pr create --title "<type>(<scope>): <desc>" --base main --label "<type>" --body-file <filled-from-template>`.
   Exception PRs: add `--label no-issue` or `--label hotfix` and complete `## Exception`.
6. **CodeRabbit**: With `.coderabbit.yaml` auto-review enabled, a first review usually starts without action. If none appears (UI/org override, rate limit), or to force an immediate pass: `gh pr comment <N> --body "@coderabbitai review"`.
7. Wait for CI: `gh pr checks <N>` until **`ci-pass`** is success (do not merge on red).
8. On `gate-policy` failure: re-run `.reinguard/scripts/check-pr-policy.sh` locally, then `gh pr edit <N> --body-file ...` or `--body` with corrected sections; add missing **type** label if needed. If you patch the body via `gh api`, follow Semantics § **PR body updates** in `.reinguard/policy/workflow--pr-discipline.md` so newlines are not corrupted.

## Output

- PR URL and number; `gh pr checks` snapshot after push.

## Guard

HS-PR-BASE, HS-PR-TEMPLATE, HS-LOCAL-VERIFY, HS-MERGE-CONSENSUS
