# Load test — real account, substantial data

Evidence for the release-checklist item _"tested on a real account with
substantial data (throttling only shows at scale)."_

## Setup

- Seeded **1,206 namespaces** in `gcp-us-central1` (1,200 `loadtest-NNNN` + the 6
  posture-test namespaces), each with a small schema and 3 documents.
- 1,206 namespaces spans **3 list pages** at the plugin's `page_size=500`, so
  pagination and the `next_cursor` loop are genuinely exercised.
- Seeded and torn down with a parallel writer (20 workers).

## Results (all zero-error)

| Scan | Work | Time | Errors |
|------|------|------|--------|
| `count(*)` on `turbopuffer_namespace` | list all 3 pages | ~1.0s | 0 |
| metadata aggregate | **1,206 metadata GETs** through `MaxConcurrency=10` | ~11.5s | 0 |
| `turbopuffer_namespace_attribute` | flatten every schema → **4,025 attribute rows** | ~11.2s | 0 |
| full Powerpipe benchmark (16 controls) | **14,886 result rows** | ~2m | 0 |

- **Pagination:** the plugin returned all 1,206 namespaces (3 pages) — no
  truncation, no duplicates.
- **Throttling:** no `429`s were returned or retried even under 1,206
  concurrent-capped metadata hydrates; turbopuffer absorbed the load and the
  `MaxConcurrency=10` cap kept the plugin polite.
- **Correctness at scale:** the benchmark's 10 alarms were exactly the 6 seeded
  misconfigurations; the 1,200 loadtest namespaces were clean, as expected.

## Reproducing

The seeder (`scripts` scratch, not committed) creates `loadtest-NNNN`
namespaces and tears them down with `--delete`. Load data was removed after the
run; the account is back to the 6 posture-test namespaces.
