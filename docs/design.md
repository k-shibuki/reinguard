# reinguard design

## Title

reinguard: a spec-driven control-plane substrate for AI agent development control systems

## One-line definition

reinguard is a spec-driven control-plane substrate that reads repo-declared
control specs, builds operational context through observation and evaluation,
and returns route, guard, and knowledge-pack outputs for the agent without
taking over semantic judgment.

## 0. What reinguard is

reinguard is a control-plane substrate for AI-agent-driven development.

More precisely, reinguard is an execution substrate that:

- reads the control specifications declared in a repository
- observes the current repository and platform state
- evaluates that state into stable operational context
- selects route candidates
- evaluates guards deterministically
- packs relevant knowledge for the current context
- returns all of the above in a stable, typed form for the agent

reinguard is not the decision-maker.

reinguard does not replace the agent's semantic judgment.

reinguard does not decide what is architecturally correct, what a review
comment truly means, or what design trade-off should be accepted.

Its role is narrower and more important:

to stabilize the recognition surface presented to the agent.

This follows the core control-system principle that the primary way to
constrain an agent is not to script its thinking, but to design the
information space in which its reasoning occurs.

## 1. Problem statement

reinguard does not solve code generation.

reinguard solves the unstable parts of AI-agent operation, especially:

- observation variance
- state-recognition variance
- missed asynchronous platform signals from CI, review bots, and pull requests
- route inconsistency
- guard omission
- poor reachability to repo-specific knowledge

In other words, reinguard does not attempt to increase model intelligence.

It attempts to stabilize the operational context in which model intelligence
is exercised.

The target failure mode is not "the model is weak."

The target failure mode is "the model is reasoning from a noisy, incomplete,
inconsistent operational surface."

## 2. Design stance

reinguard inherits and operationalizes the following control-system stance:

1. The control system is primarily an information-space design.
2. Observation should be structured, repeatable, and typed.
3. Semantics belong to repo-controlled sources of truth, not to the runtime.
4. Judgment remains with the agent.
5. Deterministic enforcement should be separated from semantic reasoning.
6. The control substrate itself must be testable software.
7. Agent-internal state is the agent's responsibility, not the substrate's.

Therefore, reinguard must be designed as a substrate, not as a workflow brain.

## 3. Architectural position

### 3.1 Conceptual split

- repo = meaning and declaration
- reinguard = execution substrate

The repository owns:

- control semantics
- state and signal definitions
- evidence schemas
- route specs
- guard specs
- knowledge manifests
- review-bot configuration
- project-specific conventions

reinguard owns:

- observation execution
- normalization
- evaluation
- deterministic checking
- operational-context assembly
- knowledge packing
- CLI and runtime delivery

The repository defines what is true.

reinguard computes what is currently the case under those definitions.

### 3.2 Invocation model

reinguard uses a pull-based invocation model.

The agent invokes `rgd` commands when it needs operational context.
reinguard does not run as a daemon, does not push events, and does not
subscribe to webhooks.

Each invocation is stateless. reinguard reads current repo and platform
state, evaluates it against config, and returns the result. No state is
carried between invocations.

This means:

- no caching layer in Phase 1
- no event subscription mechanism
- no background process management
- two invocations at slightly different times may return different results
  if the underlying repo or platform state has changed between them

This is acceptable because the agent controls invocation timing and can
re-invoke to refresh.

Future extensions (snapshot commands for audit, optional caching) are not
precluded but are not part of the initial design.

## 4. Core principles

### 4.1 reinguard strengthens observation and constraint, not reasoning

reinguard does not try to be smarter than the agent.

Its value comes from returning the same structured observation, under the
same schema, mapped into the same operational context, for the same
underlying repo state.

### 4.2 reinguard executes SSOT; it does not become SSOT

reinguard must not become the semantic authority for:

- state meanings
- transition semantics
- review semantics
- repo-specific policy
- knowledge content

Those remain in repo-side specifications (config files).

If logic migrates into Go in a way that duplicates or supersedes
repo-side declarations, the design has failed.

### 4.3 Knowledge is packed, not interpreted

reinguard may:

- index knowledge entries
- filter them
- rank them by declarative relevance
- pack them compactly
- return them with provenance

reinguard must not:

- interpret domain meaning
- derive design intent from prose
- convert knowledge into execution logic
- decide which review argument is substantively correct

### 4.4 Surface compatibility is preserved where useful

reinguard should preserve established public repo surface when beneficial.

Typical pattern:

- Go implementation is the internal substrate
- Make targets remain the external interface when already institutionalized

This preserves operational vocabulary while allowing implementation
replacement.

### 4.5 Modifiability must come from spec-driven design

The primary design requirement is not only correctness.

It is controlled change.

The following changes must be cheap:

- add or remove states
- change route selection
- change guard conditions
- change evidence composition
- change knowledge packing rules

This implies:

- declarative config files (YAML)
- runtime interpretation of config
- named evaluators for complex logic that cannot be expressed declaratively
- fixture and golden validation
- strict avoidance of hard-coded workflow branching in Go

### 4.6 Agent-internal state is the agent's responsibility

reinguard observes repository state and platform state.

reinguard does not observe, track, or depend on agent-internal state such
as local workflow phase, in-progress task markers, or agent-generated
status files.

The rationale:

- agent-generated state is self-reported, not externally verifiable
- two agents observing the same repo state would receive different
  operational context if agent-internal state were included, violating
  the stability guarantee (§22)
- fine-grained phase tracking within an implementation cycle (e.g.,
  "tests written but not yet run") is semantic judgment about the
  agent's own progress, which belongs to the agent

reinguard returns "feature branch, uncommitted changes, no PR" as the
observable facts. The agent determines what phase of its own work that
represents.

## 5. Product form

### 5.1 Deliverable

reinguard is delivered as:

- a single Go binary (`rgd`)
- a set of YAML config files (repo-side)
- a set of JSON Schemas for output validation
- a set of config templates for common integrations (e.g., bot review)

### 5.2 Binary

The binary is self-contained. It has no runtime dependency on external
interpreters (no jq, no Python, no Node).

All evaluation logic — match rules, named evaluators, observation
execution — is compiled into the binary.

### 5.3 Config files

Config files are YAML. They live in the repository under `.reinguard/`
and are version-controlled.

Config files are the repo-side source of truth for:

- state definitions and match rules
- guard definitions and admissibility rules
- route definitions and ordering
- bot review configuration
- knowledge packing rules
- observation source declarations

### 5.4 Templates

reinguard ships with built-in config templates for common patterns:

- bot review configurations (CodeRabbit, Codex, etc.)
- standard state models for GitHub-based workflows
- common guard rules (CI green, review consensus, merge readiness)

Templates are starting points. Projects override them via their own
`.reinguard/` config files.

## 6. Config structure

### 6.1 File layout
```text
.reinguard/
  reinguard.yaml            # global settings (CLI, output, schema version)
  state-rules.yaml          # state match rules + route rules
  guard-rules.yaml          # guard admissibility rules
  bots/
    coderabbit.yaml         # bot review config (template-derived)
    codex.yaml
  knowledge-manifest.yaml   # knowledge packing rules
```

### 6.2 Evaluation expression model

Config rules use a declarative match syntax with a bounded set of
operators. When a judgment requires algorithmic logic that exceeds the
match syntax, it is delegated to a named evaluator compiled into the
binary.

#### Match rules

Match rules are the primary mechanism for state classification, guard
evaluation, and route selection.
```yaml
state_rules:
  - when: { ci_status: "failure" }
    then: CIFailed
    priority: 2
    depends_on: [pull_request]

  - when: { mergeable: "CONFLICTING" }
    then: DependentChainRebase
    priority: 3
    depends_on: [pull_request]

  - when:
      review_threads_unresolved: { gt: 0 }
    then: UnresolvedThreads
    priority: 4
    depends_on: [pull_request]
```

#### Named evaluators

Complex judgments that require aggregation across multiple signals,
iterative logic, or domain-specific algorithms are compiled into Go
as named evaluators. Config references them by name.
```yaml
guard_rules:
  merge_readiness:
    evaluator: "review_consensus"
    params:
      require_ci_green: true
      require_no_unresolved: true
      require_bot_terminal: true
```

Named evaluators are a controlled extension point. The number of named
evaluators is itself a design metric: growth in this count is a signal
of potential semantics leakage and should be tracked.

### 6.3 Operator set

The match rule engine supports the following operators:

**Scalar comparison:** `eq`, `ne`, `gt`, `lt`, `gte`, `lte`

**Collection:** `in` (value in list), `contains` (list contains value)

**Existence:** `exists`, `not_exists`

**Aggregation:** `count`, `any`, `all` (over array fields)

**Logical:** `and`, `or`, `not`

Arithmetic operators are excluded by design. Computations requiring
arithmetic (e.g., summing findings counts across bots) belong in named
evaluators.

### 6.4 State-level degradation dependency

Each state rule declares which observation sources it depends on via
`depends_on`. If a declared dependency is degraded (stale, unavailable,
errored), that state candidate is suppressed and the evaluator falls
through to the next priority candidate.
```yaml
state_rules:
  - when: { ci_status: "failure" }
    then: CIFailed
    priority: 2
    depends_on: [pull_request]

  - when: { stale_branches_count: { gt: 0 } }
    then: StaleBranches
    priority: 6
    depends_on: [workflow_position]
```

This prevents false state classification when observation is incomplete.
If all candidates for a given context are suppressed, the operational
context is marked `classification_status: degraded` and a reasoning
handoff is emitted.

## 7. Responsibility layers

reinguard consists of five primary layers:

1. Observation engine
2. State evaluator
3. Route selector
4. Guard evaluator
5. Knowledge packer

These layers are deliberately asymmetric.

Observation is an engine because it interacts with sources and
normalization pipelines.

The others are evaluators, selectors, or packers because they interpret
repo-declared specs (config) rather than create meaning.

## 8. Observation engine

### 8.1 Responsibility

The observation engine executes structured observations against
available sources such as:

- local git state
- GitHub pull request, issue, and review state
- CI status
- review-bot status
- environment health
- repo-local metadata
- declared control artifacts

Its responsibilities are:

- collect source data
- normalize source variation
- emit typed results
- attach provenance
- structure partial failure
- remain read-only with respect to repo and platform state

### 8.2 Design requirements

Observation must be:

- deterministic where possible
- schema-bound
- read-only (it must not modify repository or platform state;
  observational side-effects such as API rate-limit consumption
  and audit log generation are acceptable)
- auditable
- freshness-scoped
- composable

### 8.3 Output discipline

All observation outputs should be machine-readable and stable.

Each result should carry:

- source
- collected_at
- spec_version
- parser or runtime version
- target name
- duration
- partial-failure records if any

### 8.4 Partial failure model

Observation failure is not binary.

reinguard should distinguish at least:

- unavailable
- stale
- auth_error
- transport_error
- parse_error
- unsupported
- inconsistent

These must be represented structurally and fed forward into state and
guard evaluation via the `depends_on` mechanism (§6.4), not hidden
behind string logs.

### 8.5 Observation scope boundary

The observation engine observes repository state and platform state.

It does not observe agent-internal state. Specifically:

- no reading of agent-generated phase files
- no reading of agent task markers or progress indicators
- no reading of agent session state

If a signal cannot be derived from git, GitHub API, CI, or other
external platform sources, it is outside reinguard's observation scope.

## 9. State evaluator

### 9.1 Responsibility

The state evaluator maps normalized signals into an effective state.

It reads state match rules from config (`state-rules.yaml`), applies
priority ordering, and resolves to an effective state.

Its role is to answer:

- where in the workflow the system currently is
- why that state was chosen
- what blockers are active
- what ambiguity or degradation remains

### 9.2 Evaluation mechanism

State evaluation proceeds as follows:

1. All match rules are evaluated against current observation signals.
2. For each matching rule, check `depends_on`: if any declared source
   is degraded, suppress this candidate.
3. Among unsuppressed matches, select the one with the highest priority
   (lowest priority number).
4. If no rules match or all are suppressed, emit
   `classification_status: degraded`.

When both a global state (from workflow-position signals) and a
PR-specific state (from pull-request signals) match, the evaluator
uses priority comparison to select the effective state. This replaces
the two-tier `global_state_id` / `pr_state_id` / `effective_state_id`
pattern from the prototype with a unified priority-based resolution.

### 9.3 Scope

The state evaluator may:

- aggregate signals
- apply priority rules
- resolve state precedence
- compute effective state
- emit reasons and blockers

It must not:

- invent new semantic states ad hoc
- embed repo-specific exceptions in runtime code
- perform planning
- depend on agent-internal state

### 9.4 Required outputs

State evaluation should produce:

- effective_state_id
- effective_state_label
- matched_signals
- suppressed_candidates (with suppression reasons)
- blockers
- why
- classification_status
- degradation details where applicable

## 10. Route selector

### 10.1 Responsibility

The route selector chooses route candidates from the current effective
state and related context.

Its role is not to execute workflow steps.

Its role is to answer:

- given the current operational context, what action routes are
  admissible and recommended next?

### 10.2 Why "route selector" and not "router"

The term "router" suggests active orchestration or control-flow
ownership.

reinguard should avoid implying that it drives the workflow.

It selects route candidates for the agent.

### 10.3 Scope

The route selector may:

- map state to candidate action routes
- apply ordering and prioritization
- suppress inapplicable routes
- expose route metadata
- return route prerequisites

It must not:

- execute actions
- own orchestration semantics
- hide blockers
- replace agent judgment on which route to take when multiple are valid

### 10.4 Route metadata

A route may include metadata such as:

- route id
- action card or command id
- preconditions
- related guards
- expected evidence refresh points
- knowledge-pack hints
- blocking reasons if not currently admissible

## 11. Guard evaluator

### 11.1 Responsibility

The guard evaluator performs deterministic admissibility checks.

This includes checks such as:

- merge readiness
- required review completeness
- unresolved-thread blocking
- required template or section completeness
- hard-stop violations
- unsafe action attempts

### 11.2 Position in the architecture

Guard evaluation is not semantic review.

It is deterministic constraint checking.

### 11.3 Evaluation mechanism

Guard rules follow the same two-tier model as state rules:

- Simple admissibility checks use match rules in `guard-rules.yaml`.
- Complex checks (e.g., review consensus requiring bot tier aggregation,
  findings count summation, and multi-condition AND) use named evaluators
  referenced from config.

Named evaluators for guards are compiled into the binary. Initial set:

- `review_consensus`: evaluates bot review completion, findings totals,
  thread resolution, disposition, and rereview pending state to produce
  a consensus determination. Parameterized via bot config files.
- `merge_readiness`: composes CI status, mergeable status, merge state,
  and review consensus into a safe-to-merge determination.

Additional named evaluators may be added, but each addition must be
justified. The count of named evaluators is a tracked design metric.

### 11.4 Scope

The guard evaluator may:

- evaluate declared guard specs (match rules and named evaluators)
- return pass, fail, or degraded results
- enumerate blockers
- explain which condition failed
- provide machine-readable admissibility results

It must not:

- invent policy
- reinterpret repo semantics
- silently override hard stops

### 11.5 State vs guard boundary

The boundary must remain strict:

- state = where the workflow currently is
- guard = whether a route or action is admissible
- route = which action candidates are recommended given state and guards

This separation is critical to preserving modifiability.

## 12. Knowledge packer

### 12.1 Responsibility

The knowledge packer selects and packages relevant repo-owned knowledge
for the current operational context.

Its role is not knowledge understanding.

Its role is delivery.

### 12.2 Scope

The knowledge packer may:

- load manifests
- select knowledge atoms by declarative rules
- filter by state, route, changed paths, failed checks, or review findings
- pack compact subsets
- emit references and provenance

It must not:

- interpret domain meaning
- resolve conflicting guidance semantically
- transform prose into runtime policy
- act as a planner

### 12.3 Why "packer"

Because the runtime role is to compose an agent-usable bundle.

It is narrower and safer than "resolver."

The packer should remain a packaging layer, not a semantic engine.

## 13. Ambiguity and reasoning handoff

### 13.1 Why this layer is needed

Not every situation can be deterministically collapsed into a single
immediately actionable workflow state.

There will be cases where:

- observation is incomplete
- signals conflict
- multiple states remain plausible
- the repo is in an off-nominal condition not fully covered by
  current specs
- semantic interpretation is needed before safe routing is possible

In those cases, reinguard should not pretend certainty.

### 13.2 Design rule

Ambiguity should not usually be modeled as a normal workflow state
such as implementing or CI pending.

Instead, it should be modeled as an evaluation outcome attached to
operational context.

Fields:

- `classification_status: resolved | ambiguous | degraded | unsupported`
- `reasoning_handoff_required: true | false`
- `candidate_interpretations: [...]`
- `missing_evidence: [...]`
- `suppressed_candidates: [...]`
- `reentry_contract: ...`

### 13.3 Integration with degradation model

When state-level `depends_on` declarations cause all candidates to be
suppressed (§6.4), the classification status is set to `degraded`, and
the reasoning handoff includes:

- which sources were degraded and why
- which state candidates were suppressed
- what observation refresh would resolve the ambiguity

### 13.4 Meaning

This allows reinguard to say:

- what is known
- what remains underdetermined
- what bounded reasoning task is required from the agent
- what the expected return path is after that reasoning

This preserves the substrate model:

- reinguard observes and evaluates
- the agent reasons when semantic interpretation is needed
- the agent returns to reinguard for refreshed operational context
  and continued routing

### 13.5 Re-entry contract

A reasoning handoff should include an explicit re-entry contract,
such as:

- provide selected interpretation
- request additional evidence target
- mark this route as blocked pending human decision
- continue with a narrowed candidate set

This avoids ambiguous back-and-forth and keeps the substrate-agent
boundary explicit.

## 14. Primary output: operational context

### 14.1 Definition

The central output of reinguard is operational context.

Operational context is the runtime-assembled, typed, auditable
representation of the current agent-facing control surface.

### 14.2 Contents

At minimum, operational context contains:

- observation summary
- normalized evidence references
- effective state
- blockers and why
- selected route candidates
- guard status
- knowledge pack references
- provenance and freshness metadata
- partial-failure diagnostics
- classification status
- suppressed candidates with reasons
- reasoning handoff metadata when required

### 14.3 Output format

Operational context is serialized as JSON.

A JSON Schema is published for each schema version. The schema is part
of the reinguard distribution and is versioned alongside the binary.

Agents and downstream tools may validate operational context against
the schema.

### 14.4 Output structure
```json
{
  "schema_version": "string",
  "effective_state": {
    "id": "string",
    "label": "string",
    "matched_signals": ["string"],
    "suppressed_candidates": [
      {
        "state_id": "string",
        "reason": "string",
        "degraded_source": "string"
      }
    ],
    "blockers": ["string"],
    "why": "string"
  },
  "classification_status": "resolved | ambiguous | degraded | unsupported",
  "reasoning_handoff": {
    "required": "boolean",
    "candidate_interpretations": ["string"],
    "missing_evidence": ["string"],
    "reentry_contract": "string"
  },
  "route_candidates": [
    {
      "id": "string",
      "action": "string",
      "preconditions": ["string"],
      "related_guards": ["string"],
      "knowledge_hints": ["string"],
      "blocked": "boolean",
      "blocked_reason": "string | null"
    }
  ],
  "guard_results": {
    "<guard_id>": {
      "status": "pass | fail | degraded",
      "blockers": ["string"],
      "evaluator": "string (match_rule | named_evaluator_name)"
    }
  },
  "knowledge_pack": {
    "refs": [
      {
        "path": "string",
        "relevance_tag": "string",
        "required": "boolean"
      }
    ]
  },
  "observation_summary": {
    "<source_name>": {
      "status": "ok | stale | unavailable | auth_error | transport_error | parse_error",
      "collected_at": "ISO8601",
      "duration_ms": "integer"
    }
  },
  "_meta": {
    "rgd_version": "string",
    "config_version": "string",
    "schema_version": "string",
    "timestamp": "ISO8601",
    "duration_ms": "integer"
  }
}
```

### 14.5 Operational context is not raw state

Operational context is the evaluated control surface needed for correct
agent operation. It is not merely a pass-through of observation results.

## 15. Spec model: config file specifications

### 15.1 `reinguard.yaml` (global settings)

Defines:

- schema version
- CLI output preferences (JSON formatting, verbosity)
- observation source registry (source names, endpoints, auth references)
- default freshness policies

### 15.2 `state-rules.yaml` (state + route)

Defines:

- state match rules (when/then/priority/depends_on)
- state priority ordering
- state classification metadata (labels, categories)
- route candidate rules (state-to-route mapping)
- route ordering and suppression conditions
- route metadata

State and route are co-located because they are tightly coupled: the
effective state largely determines the route candidates. Separating
them into different files would create cross-file references without
meaningful independence.

### 15.3 `guard-rules.yaml`

Defines:

- admissibility match rules
- named evaluator references with parameters
- hard stops (non-overridable guards)
- failure reasons and severity
- evaluation semantics

### 15.4 `bots/*.yaml` (bot review configuration)

Defines per bot:

- identity (login pattern, match type)
- review tier semantics (reviewed/failed/in-progress status mappings)
- rate limit and invalidation patterns
- skip patterns and skip policy (terminal_clean / terminal_blocked)
- max reviews
- required flag
- trigger mode
- commit status name

reinguard ships templates for common bots (CodeRabbit, Codex, etc.).
Projects override or extend via their own files.

### 15.5 `knowledge-manifest.yaml`

Defines:

- manifest schema
- selection rules (by state, route, changed paths, failed checks)
- relevance tags
- required vs suggested
- pack size or token-budget policy
- packing order

## 16. Adoption tiers

Adopting reinguard does not require writing all config files at once.

### Tier 1: Minimal (state evaluation only)

Required files:

- `reinguard.yaml` (global settings)
- `state-rules.yaml` (state rules only, no route rules)

Provides: `rgd state eval` — effective state classification from
git and GitHub observation.

### Tier 2: State + Guard

Add:

- `guard-rules.yaml`
- `bots/*.yaml` (if using bot review)

Provides: `rgd guard eval merge-readiness` — merge readiness and
admissibility checks.

### Tier 3: Full

Add:

- route rules in `state-rules.yaml`
- `knowledge-manifest.yaml`

Provides: `rgd context build` — full operational context with
routes and knowledge packs.

This progressive adoption path ensures that projects can start with
state evaluation alone and add layers as needed.

## 17. Concurrency model

reinguard is stateless per invocation (§3.2).

There is no shared state between concurrent invocations. If two agent
sessions invoke `rgd context build` simultaneously, each performs its
own full observation cycle independently.

This means:

- no locking
- no shared cache
- no coordination between invocations
- results may differ if underlying state changes between invocations

This is the simplest correct model for a pull-based substrate. It
avoids the complexity of cache invalidation, TTL management, and
distributed consistency.

A snapshot mechanism for audit and testing purposes (explicit
`rgd snapshot create` followed by `rgd context build --snapshot=<id>`)
is a future extension, not part of the initial design.

## 18. Repo separation principle

reinguard lives separately from any specific project repository.

The split is:

Repository side:

- `.reinguard/` config directory
- `docs/agent-control/` (project knowledge, if applicable)
- knowledge atoms
- project-specific fixtures and goldens
- Make surface where preserved

reinguard side:

- Go binary (`rgd`)
- observation engine
- match rule engine
- named evaluators
- config loaders and validators
- JSON Schema definitions
- built-in config templates
- reusable CLI implementation

This separation preserves:

- portability
- clean authority boundaries
- reusability across repositories
- prevention of semantics leakage into runtime code

## 19. Migration strategy (from bridle)

### 19.1 Prototype relationship

The `k-shibuki/bridle` repository (`docs/agent-control/`) is the
prototype that reinguard generalizes. Bridle's jq-based FSM, evidence
shell scripts, and state model are the working reference implementation.

### 19.2 Migration approach: fixture-first + live validation

1. **Extract golden fixtures**: collect bridle's jq input/output pairs
   as golden test cases for the Go implementation.
2. **Implement against goldens**: build rgd's evaluation logic to pass
   the golden test suite, ensuring behavioral compatibility.
3. **Install in bridle**: once stable, install rgd into bridle and
   replace `make evidence-fsm` with `rgd context build`.
4. **Validate in production**: run both in parallel until confidence
   is established, then remove the jq pipeline.

### 19.3 What migrates where

| Bridle artifact | reinguard destination |
|---|---|
| `global-workflow.jq` state logic | `state-rules.yaml` match rules |
| `effective-state.jq` priority resolution | `state-rules.yaml` priority field + rgd engine |
| `pull-request-readiness.jq` bot/consensus logic | `review_consensus` named evaluator + `bots/*.yaml` |
| `pull-request-readiness.jq` blocker computation | `merge_readiness` named evaluator + `guard-rules.yaml` |
| `augment-routing.jq` issue selection | `state-rules.yaml` route rules |
| `review-bots.json` | `.reinguard/bots/coderabbit.yaml` etc. |
| `state-model.md` state catalog | `state-rules.yaml` (machine-readable form) |
| `state-model.md` signal catalog | `reinguard.yaml` observation source registry |
| `evidence-schema.md` | JSON Schema for observation outputs |
| `evidence-*.sh` scripts | rgd observation engine (Go) |

### 19.4 State model migration

Bridle's 21-state model reduces to 17 states in reinguard.

Removed states (agent-internal, merged into `Implementing`):

- `ImplementationDone`
- `TestsDone`
- `QualityOK`
- `TestsPass`

These represented local development phases that depend on
agent-generated state files. reinguard observes "feature branch,
uncommitted changes, no PR" and returns `Implementing`. The agent
determines its own sub-phase.

## 20. UI and UX principles

### 20.1 Naming

Formal project name: `reinguard`

CLI name: `rgd`

### 20.2 CLI shape

Representative commands:

- `rgd observe workflow-position`
- `rgd state eval`
- `rgd route select`
- `rgd guard eval merge-readiness`
- `rgd knowledge pack`
- `rgd context build`

Or, if preserving inherited surface mappings:

- `make evidence-fsm` → `rgd context build`
- `make next` → `rgd route select` plus context output

### 20.3 UX goals

The CLI should be:

- short
- composable
- structurally legible
- JSON-first where machine output is expected
- compatible with agent invocation patterns

## 21. Provenance and auditability

Operational context must be auditable.

Every meaningful output should preserve:

- which sources were consulted
- when they were consulted
- which config version was used
- which rgd version was used
- where degradation occurred
- why a route or guard result was produced
- which state candidates were suppressed and why

Without this, reinguard may look stable while being operationally opaque.

## 22. Testability

reinguard must itself be testable software.

### 22.1 Golden test strategy

The primary validation strategy is fixture and golden driven.

Golden tests are initially sourced from bridle's jq pipeline
input/output pairs (§19.2). As reinguard evolves, new goldens are
added for cases not covered by the prototype.

### 22.2 Test surfaces

Minimum test surfaces:

- raw observation fixtures
- normalized observation goldens
- state evaluation goldens (match rule + priority resolution)
- state degradation goldens (depends_on suppression)
- route selection goldens
- guard evaluation goldens (match rule + named evaluator)
- knowledge-pack goldens
- end-to-end operational-context goldens
- ambiguity and reasoning-handoff goldens
- config validation tests (malformed config detection)
- named evaluator unit tests

### 22.3 Test contract

The test unit is not only function-level behavior.

It is the control-surface contract: given this config and these
observation inputs, reinguard must produce this operational context.

## 23. Failure modes to explicitly defend against

### 23.1 Semantics leakage into runtime

Danger:

- repo-specific meaning slowly migrates into Go conditionals

Defense:

- strict config/runtime boundary
- named evaluator count tracking as a design metric
- code-review rule that semantic exceptions must be pushed back into
  config
- golden tests that verify config-only changes produce expected
  behavior changes

### 23.2 Knowledge packer inflation

Danger:

- the packer becomes a planner or semantic interpreter

Defense:

- packer remains manifest-driven and packaging-only

### 23.3 State and guard collapse

Danger:

- blockers are encoded inconsistently across state and guard layers

Defense:

- maintain the strict boundary:
  - state = position
  - guard = admissibility
  - route = candidate selection

### 23.4 Hard-coded workflow branching

Danger:

- Go switch statements become the de facto workflow engine

Defense:

- state, route, and guard logic are driven by config match rules
- complex logic is isolated in named evaluators with stable interfaces
- golden tests enforce that behavior changes require config changes

### 23.5 False certainty under ambiguity

Danger:

- reinguard emits a single route despite incomplete or conflicting
  evidence

Defense:

- state-level `depends_on` declarations with degradation suppression
- explicit classification status
- bounded reasoning handoff
- re-entry contract
- degraded or ambiguous outputs rather than fabricated certainty

### 23.6 Named evaluator proliferation

Danger:

- the number of named evaluators grows unchecked, effectively moving
  all logic into Go and leaving config as a thin wrapper

Defense:

- named evaluator count is a tracked metric
- each new named evaluator requires explicit justification
- evaluator interfaces must be stable and parameterizable
- if a pattern recurs across evaluators, it should be generalized
  into a match-rule operator instead

## 24. Non-goals

reinguard must not do the following:

- decide architectural direction
- decide whether a review finding is semantically valid
- justify exception handling on its own
- internalize repo-specific design judgment
- convert knowledge prose into embedded logic
- think on behalf of the agent
- become a general workflow orchestrator
- track or depend on agent-internal state

reinguard is not an autonomous planner.

reinguard is not a substitute for the agent.

reinguard is not a project-management system.

reinguard is not a code-generation framework.

It is a control-plane substrate.

## 25. Success criteria

reinguard succeeds if:

1. Repo-declared control specs (config files) remain the semantic
   authority.
2. The runtime builds stable operational context from those specs.
3. State, route, and guard changes are made primarily in config and
   tests, not code.
4. Knowledge remains repo-owned and is only packed, not interpreted.
5. Existing repo operational vocabulary can be preserved where useful.
6. Different agents or sessions observing the same repo state receive
   materially equivalent operational context.
7. The substrate improves control reliability without taking over
   judgment.
8. Ambiguous situations are surfaced explicitly rather than hidden
   behind false determinism.
9. The named evaluator count remains small and each evaluator is
   justified.
10. A project can adopt reinguard progressively, starting with state
    evaluation alone.

## 26. Final summary

reinguard is a guarded, spec-driven control-plane substrate.

It is delivered as a single Go binary (`rgd`) with YAML config files
and JSON Schema output contracts.

It reads repo-declared control specifications, executes structured
observation, evaluates effective state through priority-ordered match
rules, selects candidate routes, checks guards deterministically
(via match rules and named evaluators), and packs relevant knowledge
into an operational context suitable for agent reasoning.

It does not observe agent-internal state.

It does not become the semantic authority.

It does not replace the agent's judgment.

It does not act as a workflow brain.

Its purpose is narrower:

to make the agent's operational context stable, typed, auditable,
and reusable.

That is the substrate layer missing between declarative repo control
systems and agent reasoning.