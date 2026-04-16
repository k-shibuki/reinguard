---
id: review-github-thread-api
description: GitHub REST vs GraphQL for PR review thread resolution state (isResolved) and enumeration
triggers:
  - isResolved
  - unresolved thread count
  - review thread status
  - GraphQL reviewThreads
  - thread resolution API
  - gh api graphql
when:
  op: eq
  path: github.pull_requests.pr_exists_for_branch
  value: true
---

# GitHub review thread API (REST vs GraphQL)

## Constraint

GitHub REST endpoints:

- `GET /repos/{owner}/{repo}/pulls/{pull_number}/comments`
- `GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews`

return individual review comments and reviews. They do **not** expose
per-thread resolution state. There is no `isResolved` (or equivalent)
field on those resources for counting or listing **unresolved**
conversation threads.

To enumerate threads and read `isResolved`, use the **GraphQL** API:
`repository.pullRequest.reviewThreads`.

When available, prefer `rgd` commands for review thread operations:

- Read threads with `rgd observe github reviews --view inbox` or
  `rgd context build --compact` for the structured `review_inbox` read model.
- Reply and resolve with `rgd review reply-thread` and
  `rgd review resolve-thread` for the covered happy-path transport.

Use the raw GraphQL flow below as the fallback and truth source for cases not
yet surfaced by `rgd`.

## Enumerate threads with resolution state

Use `gh api graphql` and paginate while `pageInfo.hasNextPage` is true:

```bash
gh api graphql -f query='
query($owner:String!, $name:String!, $number:Int!, $cursor:String) {
  repository(owner:$owner, name:$name) {
    pullRequest(number:$number) {
      reviewThreads(first:100, after:$cursor) {
        nodes { id isResolved isOutdated }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}' -f owner=OWNER -f name=REPO -F number=N
```

Count `nodes` where `isResolved` is `false` for unresolved threads.
Repeat with `after: $cursor` set to `pageInfo.endCursor` until
`hasNextPage` is false.

## Branch protection

Threads with `isOutdated: true` can still block merge when Branch
Protection requires conversation resolution; do not assume outdated
threads are ignorable for merge readiness.

## Related

- `.reinguard/policy/review--consensus-protocol.md` — disposition,
  resolve mutation, completeness invariant
- `.reinguard/knowledge/review--bot-operations.md` — REST channels for
  reviews and inline comments (without per-thread `isResolved`)
