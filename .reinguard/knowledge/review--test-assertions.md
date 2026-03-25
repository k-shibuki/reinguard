---
id: review-test-assertions
description: Test assertion quality, failure messages, and boundary coverage
triggers:
  - test quality
  - assertion strength
  - given when then
  - table-driven
---

# Test Assertion Quality

## Rule 1: Assert the expected failure cause

For negative tests, verify the intended failure mode, not just `err != nil`.

Examples:
- missing file path should mention the missing file
- malformed JSON should be a parse/syntax error category

## Rule 2: Assert concrete resolved outcomes

For success tests, verify domain outcomes (e.g. `kind`, `state_id`, `route_id`)
instead of only checking that some JSON exists.

## Rule 3: Keep Given/When/Then in edited tests

When creating or substantially editing non-trivial tests:
- add concise Given/When/Then comments
- keep test intent readable in code review

## Rule 4: Add boundary checks when changing contracts

If behavior around empty/missing inputs changes:
- add at least one explicit boundary test
- assert both key presence and value shape/type
