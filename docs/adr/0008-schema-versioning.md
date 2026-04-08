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

5. **CLI SSOT:** Normative documentation for flags, exit codes, stdin/stdout
   contracts, and the command tree lives in **`docs/cli.md`**, not in this
   ADR. This ADR references that file as the operational contract for
   machine-readable CLI I/O.

6. **Versioned repository files:** Any repository-owned file whose embedded JSON Schema requires a top-level `schema_version` (including `reinguard.yaml`, optional `labels.yaml`, `knowledge/manifest.json`, policy and control catalogs where present, and each `rules` bundle under `.reinguard/control/{states,routes,guards}/`) participates in the **same** synchronized semver line as `pkg/schema`’s `CurrentSchemaVersion` (ADR-0011 layout). Adding or changing such a file without bumping the shared contract is a schema change.

7. **Per-file enforcement:** `config.Load` / `rgd config validate` reject **major** mismatches for **each** declared `schema_version`. **Same-major** skew (older or newer minor/patch) emits **warnings on stderr** per file, not only for the root `reinguard.yaml`. Control rule YAML documents validate against `rules-document.json`, which requires a top-level `schema_version` alongside `rules`.

## Consequences

- **Easier**: Progressive adoption; semver communicates breaking vs
  additive change
- **Easier**: One version number to discuss in support and changelogs
- **Harder**: Any output-only contract change may force a version bump
  that config files also declare, even when YAML structure is unchanged
- **Harder**: Best-effort parsing requires disciplined deprecation and
  thorough tests to avoid silently accepting broken configs
- **Harder**: Bumping the shared contract (for example to `0.7.0`) touches every versioned Semantics file that declares `schema_version`, not only the root config
