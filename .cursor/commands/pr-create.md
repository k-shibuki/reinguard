# pr-create

## Context

- `.github/PULL_REQUEST_TEMPLATE.md` (body SSOT — **HS-PR-TEMPLATE**)
- `.reinguard/policy/commit--format.md` (branch naming; Cursor: `commit-format.mdc`)
- `tools/check-pr-policy.sh` (local pre-flight mirroring `check-policy` CI)

**Already in context** (do not re-list): `reinguard-bridge.mdc` (HS-*, catalogs), `workflow-policy.mdc` (exceptions, command separation).

**Pre-flight:** on feature branch, `git status` clean; push latest commits.

## Act

1. Push: `git push -u origin HEAD` (after **HS-LOCAL-VERIFY**).
2. **Pre-flight PR policy** (before `gh pr create`): fill the template into a file, then run from repo root:

   ```bash
   bash tools/check-pr-policy.sh \
     --title "<same-as-gh-pr-create>" \
     --body-file <filled-from-template> \
     --label "<type>" \
     --base main
   ```

   Fix any reported errors so `check-policy` CI does not fail on template/labels/title/base.
3. Create PR targeting **main** only:
   `gh pr create --title "<type>(<scope>): <desc>" --base main --label "<type>" --body-file <filled-from-template>`.
   Exception PRs: add `--label no-issue` or `--label hotfix` and complete `## Exception`.
4. Trigger CodeRabbit: `gh pr comment <N> --body "@coderabbitai review"`.
5. Wait for CI: `gh pr checks <N>` until **`ci-pass`** is success (do not merge on red).
6. On `check-policy` failure: re-run `tools/check-pr-policy.sh` locally, then `gh pr edit <N> --body-file ...` or `--body` with corrected sections; add missing **type** label if needed.

## Output

- PR URL and number; `gh pr checks` snapshot after push.

## Guard

HS-PR-BASE, HS-PR-TEMPLATE, HS-LOCAL-VERIFY, HS-MERGE-CONSENSUS
