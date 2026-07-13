# Changelog

All notable changes to this plugin are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.0.3 - 2026-07-13

### Added

- `turbopuffer_namespace.prefix` — an explicit optional key column that pushes
  down to the list API's `?prefix=` filter (e.g. `where prefix = 'prod-'`),
  mirroring the `query` column pattern in `aws_resource_explorer_search`.
  Requested in Hub review
  ([#13](https://github.com/somoore/steampipe-plugin-turbopuffer/issues/13)).
  The existing `id =` / `id like 'abc%'` pushdown optimization is unchanged.

## v0.0.2 - 2026-07-10

Changes from the Steampipe Hub review
([#13](https://github.com/somoore/steampipe-plugin-turbopuffer/issues/13)):

### Changed

- Dropped the `akas` column from `turbopuffer_namespace`,
  `turbopuffer_namespace_attribute` and `turbopuffer_document` — an aka is a
  cloud-plugin convention (ARNs and the like) with no turbopuffer equivalent.
- `turbopuffer_document.id` now renders integer document ids explicitly
  (turbopuffer ids are `string | int64`), so they can never appear in
  scientific notation or lose precision. Verified live against ids beyond
  2^53.
- `turbopuffer_document.attributes` no longer duplicates the document id —
  the copy inside the JSON round-tripped through float64 downstream and
  could corrupt integer ids above 2^53; the `id` column is the exact value.
- Standardized the `region` column description across all tables.
- Narrowed the Hub category to `saas` (turbopuffer is a hosted search
  service, not an AI company) and added `engines` / `og_image` front matter
  to `docs/index.md`.
- Documented the 100-document scan cap on `turbopuffer_document`.

## v0.0.1 - 2026-07-09

First release. A read-only, region-scoped Steampipe plugin that exposes a
turbopuffer organization's data-plane API as SQL, built for vector-search
security and posture use cases.

### Added

- **Tables**
  - `turbopuffer_namespace` — namespaces per region, with size, activity,
    encryption, index status, and full attribute schema from the metadata API.
  - `turbopuffer_namespace_attribute` — one row per schema attribute, with
    `filterable`, full-text/regex/glob/fuzzy, and vector/sparse index flags.
  - `turbopuffer_document` — targeted document lookups within a namespace
    (vectors always excluded), for canary checks and small samples.
  - `turbopuffer_namespace_recall` — on-demand recall evaluation (ANN vs.
    exhaustive ground truth); runs real searches.
  - `turbopuffer_region` — configured regions and their endpoints, a join
    anchor for residency queries.
- **Connection config** — `api_key` (or the `TURBOPUFFER_API_KEY` environment
  variable) and a `regions` list; the plugin fans out across all configured
  regions via a query matrix.
- **Resilience** — 404s ignored as skips, 429/5xx retried with backoff, and
  per-namespace metadata hydration capped for polite concurrency.

### Notes

- The public turbopuffer API is data-plane only, so there is no
  `turbopuffer_api_key` or billing/RBAC table.
- Endpoint paths and response fields were verified against the official
  `turbopuffer-go/v2` client and the turbopuffer OpenAPI spec, then confirmed
  against a live account (load-tested to 1,200+ namespaces).
