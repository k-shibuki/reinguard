# Phase 1 issue bodies (templates)

Markdown bodies for the Phase 1 Issue design plan live in `bodies/issue-{N}.md` where **N** is the **plan** sequence (2–22), not necessarily the GitHub issue number.

After bootstrap Issue `#1`, these were filed on GitHub as **`#3`–`#23`** (offset +1):

| Plan seq | GitHub issue | Title prefix |
|----------|--------------|--------------|
| 2 | 3 | docs(adr): ADR-0009 … |
| 3 | 4 | chore(test): Go test strategy … |
| … | … | … |
| 22 | 23 | chore: Dogfooding … |

To refresh an issue body from disk:

```bash
gh issue edit <number> --body-file tools/phase1-issues/bodies/issue-<plan-seq>.md
```

Use `<number> = <plan-seq> + 1` for the batch created after `#1` only.
