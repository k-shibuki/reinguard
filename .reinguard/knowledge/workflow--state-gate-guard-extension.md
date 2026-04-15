---
id: workflow-state-gate-guard-extension
description: Checklist when adding or changing FSM states, routes, runtime gates, guards, knowledge surfacing, and tests
triggers:
  - state gate guard extension
  - add state_id
  - add runtime gate
  - FSM wiring
  - workflow_fsm_test
  - control states yaml
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# State / Gate / Guard extension checklist

**Durable rules** live in [ADR-0013](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) and [ADR-0014](../../docs/adr/0014-runtime-gate-artifacts.md). This atom is the **operational** checklist for repository authors.

## Adding or changing a `state_id`

1. [ ] Update `.reinguard/control/states/*.yaml` — priorities (lower wins); residual vs refined ordering.
2. [ ] Update ADR-0013 § State catalog and any route/Adapter notes.
3. [ ] Update `.reinguard/control/routes/*.yaml` if the primary `route_id` or matching rules change.
4. [ ] Update `.reinguard/procedure/*.md` `applies_to.state_ids` / `applies_to.route_ids` for affected procedures; run `rgd config validate` (no Adapter “heuristic table”; `rgd-next` derives routing from front matter).
5. [ ] Confirm `.cursor/commands/rgd-next.md` still points at substrate + Semantics only (no duplicated mapping table).
6. [ ] Add or extend `internal/rgdcli/workflow_fsm_test.go` (or equivalent) for non-obvious resolution.

## Adding or changing a runtime **gate** (`gate_id`)

1. [ ] Document producer procedure(s) (`rgd gate record <gate-id>` after verification) and consumer(s) (`rgd gate status`, FSM `gates.<gate-id>.*`). On-disk files: `.reinguard/local/gates/<gate-id>.json` (gitignored).
2. [ ] Update ADR-0014 extension contract if semantics are new or changed.
3. [ ] Update `.reinguard/control/states/*.yaml` / `routes/*.yaml` if FSM rules reference `gates.<gate-id>`.
4. [ ] Update ADR-0013 § State catalog / procedure mapping notes if a new `state_id` or primary procedure changes.
5. [ ] Add CLI/FSM tests for `pass` vs `stale` / `missing` behavior if resolution depends on the gate.

## Adding or changing **guards** (`guard eval`)

1. [ ] Declarative rules: `.reinguard/control/guards/*.yaml` for `guard_id` wiring.
2. [ ] Built-in behavior (e.g. `merge-readiness`): [docs/cli.md](../../docs/cli.md) § `rgd guard eval` / `merge-readiness` — keep CLI SSOT aligned with code.
3. [ ] Distinguish **guard output** from **FSM `state_id`** in ADR-0013; do not imply equivalence without explicit state rules.

## Knowledge and surfacing

1. [ ] New or changed operational guidance: add/edit `.reinguard/knowledge/*.md` with `id`, `description`, `triggers`, **`when`** (required).
2. [ ] Run `rgd knowledge index` and commit `.reinguard/knowledge/manifest.json`.
3. [ ] Prefer `rgd context build` to validate `when` against merged `state.*` and `gates.*` (see [docs/cli.md](../../docs/cli.md) § `rgd context build`).

## Validation and tests (run before push)

1. [ ] `rgd config validate` — control YAML, procedure `applies_to` mapping checks, manifest freshness, `when` static checks.
2. [ ] `go test ./... -race` (and `go vet`, `golangci-lint` per project policy) when Go or tests change.

**Targeted tests (when touched by the change):**

| Area | Typical packages / files |
|------|---------------------------|
| FSM state / route resolution | `internal/rgdcli/workflow_fsm_test.go`, `internal/rgdcli/rgdcli_state_test.go`, `internal/rgdcli/rgdcli_route_test.go` |
| Runtime gates (`rgd gate`) | `internal/rgdcli/rgdcli_gate_test.go` |
| Built-in guards (`merge-readiness` etc.) | `internal/rgdcli/rgdcli_guard_test.go`, `internal/guard/eval_test.go`, `internal/guard/merge_test.go` |
| `context build` / knowledge filter | `internal/rgdcli/rgdcli_context_test.go`, `internal/rgdcli/rgdcli_knowledge_test.go` |

## Knowledge surfacing checks

1. [ ] After changing a knowledge entry’s `when` or `triggers`, run `rgd knowledge index` and commit `manifest.json`.
2. [ ] Smoke: `rgd context build` on a checkout (or fixture via `--observation-file`) and confirm expected `knowledge.entries` include or exclude the edited atom per `when`.

## Policy vs knowledge

- **Policy** (`.reinguard/policy/`) — normative HS-* and obligations; cite from procedures when needed.
- **Knowledge** (this tree) — judgment aids and checklists; **not** a substitute for ADR or `docs/cli.md` SSOT.

## Suggested implementation order (handoff)

1. ADR-0013 / ADR-0014 — durable rules and catalog rows.
2. `.reinguard/control/states/*.yaml` and `routes/*.yaml` — priorities and `when` clauses.
3. Procedures — `applies_to` updates; Adapter commands reference substrate + procedure files only.
4. Knowledge — new/changed atoms; `rgd knowledge index`; manifest committed.
5. `docs/cli.md` — if built-in guard or CLI behavior changes.
6. Go tests — `workflow_fsm_test` and package tests per the table above; then full `go test ./... -race`.
7. `rgd config validate` — final gate before push.
