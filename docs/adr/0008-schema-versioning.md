# ADR-0008: Schema versioning: synchronized semver with best-effort compatibility

## Context

reinguard publishes machine-readable contracts for:

- **Input** — repository configuration (YAML under a conventional
  directory)
- **Output** — operational context (JSON consumed by agents and tools)

Both need **schema validation** and a **versioning story** so that:

- The binary, configuration, and downstream consumers evolve together
- Older repositories can adopt new releases without a single atomic
  flag day

Alternatives considered:

| Dimension | Options |
|-----------|---------|
| Version format | Semantic versioning vs calendar versioning vs integer |
| Input vs output | One version id for both vs independent versions |
| Compatibility | Strict (mismatch = error) vs N-1 support vs best-effort |

**Strict** lockstep minimizes surprise but blocks gradual rollout.
**Independent** input and output versions add bookkeeping overhead.
**Calendar** versions do not encode compatibility expectations as clearly
as semver for API-shaped artifacts.

## Decision

1. **Semantic versioning** (`MAJOR.MINOR.PATCH`) for the schema contract
   identified by `schema_version` (or equivalent field in global
   settings).

2. **Synchronized contracts:** A single `schema_version` value governs
   **both** input configuration schemas and output operational context
   schema for a given release line. Revisions that affect only one surface
   still bump the shared version when the contract file set changes.

3. **Best-effort compatibility** when loading configuration:
   - Older declared versions: attempt to parse; emit **warnings** for
     deprecated fields
   - Unknown fields: **warn** and ignore when safe; if interpretation
     would be unreliable, **fail** with a clear validation message
   - Strict lockstep upgrades remain available: repositories bump
     `schema_version` when they intentionally migrate

4. **Tooling:** A `config validate` command validates configuration
   against published input schemas. Schemas ship **inside** the binary and
   can be **exported** to disk for editor integration and CI checks.

## Consequences

- **Easier**: Progressive adoption; semver communicates breaking vs
  additive change
- **Easier**: One version number to discuss in support and changelogs
- **Harder**: Any output-only contract change may force a version bump
  that config files also declare, even when YAML structure is unchanged
- **Harder**: Best-effort parsing requires disciplined deprecation and
  thorough tests to avoid silently accepting broken configs
