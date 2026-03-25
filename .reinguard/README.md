# .reinguard

This directory holds **repository-local** reinguard configuration and
knowledge Markdown under `knowledge/`. The file `knowledge/manifest.json` is
**generated** from knowledge front matter by `rgd knowledge index` (ADR-0010).

Normative architecture decisions live in `docs/adr/`, not here. These files
describe how *this* repository configures `rgd` for its own workflows.

## Authoring knowledge

Each knowledge file is a Markdown file under `knowledge/` with a **YAML front
matter** block (delimited by `---`).

### Required front matter fields

| Field | Type | Rule |
|-------|------|------|
| `id` | string | Unique across all entries; derived from the file name (e.g. `ci-permissions-and-gates`) |
| `description` | string | One-line summary of the guidance |
| `triggers` | list of strings | Non-empty; keywords for `rgd knowledge pack --query` filtering |

### File body

- **One concern per file** — keep each file atomic.
- **Stable guidance** — prefer durable rules over PR-specific or ephemeral data.
  Statistics snapshots and evidence-only data do not belong here.
- **Free-form Markdown** — use headings, lists, code blocks as needed.

### Workflow after editing

1. Create or edit the `.md` file with proper front matter.
2. Run `rgd knowledge index` to regenerate `knowledge/manifest.json`.
3. Run `rgd config validate` to verify freshness and schema compliance.
4. Commit both the `.md` file and the updated `manifest.json`.

For detailed retrieval flow and operational procedures, see
[`knowledge--ops.md`](knowledge/knowledge--ops.md).

## File naming convention

Pattern: **`<prefix>--<topic>.md`**

- `--` (double hyphen) separates prefix from topic.
- Within topic, use `-` (single hyphen) for word separation.

The prefix identifies the **semantic domain** aligned with rgd's module
structure. It tells agents *what the knowledge is about* and *when to
consult it*.

### rgd module-aligned prefixes

Derived from the CLI command tree (`docs/cli.md`) and internal package
structure.

| Prefix | rgd module / package | When to consult |
|--------|---------------------|-----------------|
| `observation--` | `observe`, `observation` — signal collection, validation, flattening | Changing observe / state / route code |
| `resolve--` | `resolve` — state/route priority resolution, outcomes (ADR-0004/0007) | Changing state eval / route select logic |
| `guard--` | `guard` — guard evaluation (e.g. merge-readiness) | Changing guard implementation or config |
| `knowledge--` | `knowledge` — manifest generation, validation, search (ADR-0010) | Authoring or maintaining knowledge files |
| `config--` | `config`, `configdir` — config loading, schema validation | Changing reinguard.yaml / rules/*.yaml |
| `schema--` | `pkg/schema` — embedded JSON Schemas, versioning (ADR-0008) | Changing schema contracts |
| `cli--` | `rgdcli` — CLI output shape, flags, command structure | Writing CLI output code or its tests |
| `match--` | `match` — `when` expression evaluation (ADR-0002) | Changing match expression logic |

### Cross-cutting prefixes

For knowledge that spans multiple rgd modules.

| Prefix | Domain | When to consult |
|--------|--------|-----------------|
| `testing--` | Go test quality and conventions (all packages) | Writing or reviewing tests |
| `ci--` | GitHub Actions, branch protection, CI gates | Changing `.github/workflows/` |
