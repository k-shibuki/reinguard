# issue-create — create a GitHub Issue (agent procedure)

Adapter layer: reference Semantics paths only; do not duplicate policy body text (ADR-0001).

## 1. Choose template

- **Task** — implementation or chore work (Conventional Commits title + type label). Starting point: `.github/ISSUE_TEMPLATE/task.yml`.
- **Epic** — phased parent work; label `epic`. Starting point: `.github/ISSUE_TEMPLATE/epic.yml`.

## 2. Build the body file

Construct a Markdown file for `--body-file` with the sections required by the template (see `.reinguard/policy/workflow--pr-discipline.md` § Recommended Issue sections).

Use real newlines (same constraints as PR bodies in `.reinguard/policy/workflow--pr-discipline.md` § PR body updates).

## 3. Choose labels

SSOT: `.reinguard/labels.yaml`.

- **Task**: exactly one **type** label (`feat`, `fix`, …) from `categories.type`.
- **Epic**: label **`epic`** only (no type label).

## 4. Pre-flight validation

```bash
bash .reinguard/scripts/check-issue-policy.sh \
  --title "<title>" \
  --body-file /path/to/issue-body.md \
  --label <feat|…|epic> \
  [--template task|epic]
```

Fix any reported errors and re-run until it prints `Issue policy pre-flight OK.`

## 5. Create the Issue

```bash
gh issue create --title "<title>" --body-file /path/to/issue-body.md --label "<label>"
```

For multiple labels (if ever needed): repeat `--label` or use `gh issue edit` after create.

## Related

- `.reinguard/scripts/check-issue-policy.sh`
- `.reinguard/policy/workflow--pr-discipline.md`
- `.reinguard/procedure/implement.md` (branch naming uses the same type vocabulary as `labels.yaml`)
