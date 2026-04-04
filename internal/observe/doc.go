// Package observe implements the pull-based observation engine: registered providers
// collect externally observable signals into a merged map, with non-fatal diagnostics
// and a degraded flag when any provider fails or returns partial data.
//
// # Inputs and outputs
//
// Configuration selects which providers run (ProviderSpec in the config package; see
// ADR-0008 schema versioning and ADR-0009 provider registry). Collect merges each
// provider’s Fragment under a top-level key equal to the provider ID (for example
// "git", "github"). Diagnostics aggregate per-provider messages; degraded is true if
// any provider returned an error, flagged partial data, or returned Degraded on its
// fragment.
//
// # Error semantics
//
// Engine.Collect returns an error only for nil receiver, nil config root, or unknown
// provider IDs before collect. Per-provider failures become diagnostics and set degraded
// instead of failing the whole run. LoadSignalsFileOrCollect returns I/O and JSON
// errors when reading a saved observation file; otherwise it delegates to Collect with
// the same rules.
//
// # ADR traceability
//
// ADR-0003 (pull-based, stateless CLI observation), ADR-0005 (no agent-internal
// observation paths), ADR-0006 (GitHub via gh for the GitHub provider), ADR-0009
// (provider registry and configuration).
//
// # Local-first vs remote-only boundaries
//
// Local-first (subprocess git / parsed remote; no GitHub API): GitHub repository identity
// (owner/name) from remote.origin.url when it targets github.com; branch, HEAD, working tree,
// stash, upstream ahead/behind (git provider).
//
// Remote-only (GitHub REST/GraphQL via gh auth token per ADR-0006): issue counts, PR data,
// CI combined status, review threads, bot enrichment, mergeability.
//
// When remote facets fail (auth, network, sandbox blocks), the engine keeps local-first
// signals (e.g. signals.github.repository) and attaches diagnostics; degraded is set.
package observe
