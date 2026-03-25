# pr-merge

## Context

- `.reinguard/policy/review--consensus-protocol.md` (disposition, thread resolution, CodeRabbit gate)

**Already in context**: `reinguard-bridge.mdc` (HS-*).

**Merge readiness (substrate):** structured check before merge. From repo root with `rgd` on PATH:

```bash
rgd observe > /tmp/obs.json
rgd guard eval --observation-file /tmp/obs.json merge-readiness
```

Interpret JSON: `"ok": true` only when CI success, zero unresolved review threads (aggregate signal), and clean working tree — see [docs/cli.md](../../docs/cli.md) § `merge-readiness`.

Optional full pipeline: `rgd context build` (observe → state → route → guard → knowledge entries).

**Cross-check:** `gh pr checks <N>` (especially **`ci-pass`**) and `gh pr view <N>` for mergeable state and branch protection (guard may not mirror every GitHub rule).

## Act

1. Confirm: guard / `gh` agree CI is green, PR policy satisfied, threads resolved if required.
2. Merge: `gh pr merge <N> --squash` or `--merge` per [.github/CONTRIBUTING.md](../../.github/CONTRIBUTING.md) and maintainer convention for this repo.
   Do **not** use `--admin`. Do **not** merge with failing checks.

## Output

- Confirm merged: `gh pr view <N> --json state -q .state`

## Guard

HS-CI-MERGE, HS-REVIEW-RESOLVE, HS-MERGE-CONSENSUS
