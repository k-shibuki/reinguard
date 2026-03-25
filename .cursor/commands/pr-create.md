# pr-create

## Reads

- `workflow-policy.mdc` (exceptions)
- `.github/PULL_REQUEST_TEMPLATE.md` (body SSOT — `HS-PR-TEMPLATE`)
- `commit-format.mdc` (branch naming)
- `agent-safety.mdc` (`HS-PR-BASE`, `HS-MERGE-CONSENSUS`)

## Sense

- On feature branch: `git status` clean; push latest commits.

## Act

1. Push: `git push -u origin HEAD` (after `HS-LOCAL-VERIFY`).
2. Create PR targeting **main** only:  
   `gh pr create --title "<type>(<scope>): <desc>" --base main --label "<type>" --body-file <filled-from-template>`.
   Exception PRs: add `--label no-issue` or `--label hotfix` and complete `## Exception`.
3. Trigger CodeRabbit: `gh pr comment <N> --body "@coderabbitai review"`.
4. Wait for CI: `gh pr checks <N>` until **`ci-pass`** is success (do not merge on red).
5. On `check-policy` failure: `gh pr edit <N> --body-file ...` or `--body` with corrected sections; add missing **type** label if needed.

## Output

- PR URL and number; `gh pr checks` snapshot after push.

## Guard

- `HS-PR-BASE`, `HS-PR-TEMPLATE`, `HS-LOCAL-VERIFY`
- `HS-MERGE-CONSENSUS`: do not enable auto-merge while bot review pending / threads unresolved
