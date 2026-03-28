---
id: testing-mock-strategy
description: "Mock and subprocess strategy — httptest for GitHub API, hermetic git, gh stub"
triggers:
  - mock
  - GitHub API
  - httptest
  - gh stub
  - test subprocess
  - hermetic git
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Mocks and subprocesses

- **HTTP**: `net/http/httptest` for GitHub API shapes; do not hit real API
  in default tests.
- **Git**: temporary repositories with `git init` or mocked command
  runners; avoid depending on the developer's global git config.
- **`gh`**: inject fake token or stub the command runner in unit tests;
  integration tests may use build tag `integration`.

## Related

- `.reinguard/knowledge/testing--strategy.md` — general test strategy
- `.reinguard/knowledge/testing--setup-error-handling.md` — fail-fast setup
