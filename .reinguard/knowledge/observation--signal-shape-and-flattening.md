---
id: observation-signal-shape-and-flattening
description: Observation signal shape, signals field validation, and dotted-path flattening
triggers:
  - signals field
  - observation-file validation
  - state.kind
  - flatten dotted keys
---

# Observation Signal Shape and Flattening

## Rule 1: Reject invalid observation JSON shape

When loading observation documents from file:
- require top-level `signals`
- require `signals` to be an object

Do not silently coerce missing/invalid `signals` into `{}` for guard/state/route
evaluation paths. Return validation errors instead.

## Rule 2: Dotted-path rules require flattened keys

If rules match paths like:
- `state.kind`
- `state.state_id`

then nested maps must be flattened into dotted keys before resolution.

Implementation pattern:
- merge `flattenSignals(map[string]any{"state": nestedState})` into the input map

## Rule 3: Keep route and context pipelines consistent

If route selection logic is used in both:
- `route select --state-file`
- `context build`

apply the same flattening contract in both pipelines to avoid divergence.
