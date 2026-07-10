---
organization: somoore
category: ["saas"]
icon_url: "/images/plugins/somoore/turbopuffer.svg"
brand_color: "#FB915F"
display_name: "turbopuffer"
short_name: "turbopuffer"
description: "Steampipe plugin to query turbopuffer namespaces, schemas, documents and recall — the inventory layer for vector-search security posture."
og_description: "Query turbopuffer with SQL! Open source CLI. No DB required."
og_image: "/images/plugins/somoore/turbopuffer-social-graphic.png"
engines: ["steampipe", "sqlite", "postgres", "export"]
---

# turbopuffer + Steampipe

[turbopuffer](https://turbopuffer.com) is a fast, object-storage-native search engine (vector + full-text). [Steampipe](https://steampipe.io) exposes APIs as SQL tables.

Together: your entire vector-search estate becomes queryable — and auditable.

```sql
select id, region, approx_row_count, encryption_mode, updated_at
from turbopuffer_namespace
order by approx_logical_bytes desc;
```

```
+---------------+------------------+------------------+------------------+---------------------+
| id            | region           | approx_row_count | encryption_mode  | updated_at          |
+---------------+------------------+------------------+------------------+---------------------+
| prod-acme     | gcp-us-central1  | 18442210         | customer_managed | 2026-07-05 11:02:41 |
| prod-globex   | gcp-us-central1  | 9120022          |                  | 2026-03-14 09:55:07 |
+---------------+------------------+------------------+------------------+---------------------+
```

## Documentation

- **[Table definitions & examples →](tables)**
- **[Example queries →](example_queries.md)** — a guided getting-started tour

## Quick start

Install the plugin (once published; for now, build locally with `make install`):

```bash
steampipe plugin install somoore/turbopuffer
```

Configure your connection in `~/.steampipe/config/turbopuffer.spc`:

```hcl
connection "turbopuffer" {
  plugin  = "somoore/turbopuffer"
  api_key = "tpuf_..."                # or TURBOPUFFER_API_KEY
  regions = ["gcp-us-central1"]       # list ALL regions your org can reach
}
```

Run a query:

```bash
steampipe query "select id, region from turbopuffer_namespace"
```

## Credentials

| Item        | Description |
|-------------|-------------|
| Credentials | A turbopuffer API key, created in the dashboard under **API Keys**. |
| Permissions | The plugin is read-only: it issues namespace lists, metadata reads, and (only when you explicitly qual them) document queries and recall evaluations. Use a read-scoped key, but note it must include the **list namespaces** permission — turbopuffer issues read keys without it, and those fail the `turbopuffer_namespace` table with a 403. |
| Radius      | Each connection scans one organization across the `regions` you list. Regions are separate endpoints; unlisted regions are invisible. |
| Resolution  | 1. `api_key` in the connection config. 2. `TURBOPUFFER_API_KEY` environment variable. |

## Multiple Connections

Each `turbopuffer` connection scans one organization. To query several at once — multiple orgs, or a prod/staging split — define a connection per credential and group them with an [aggregator](https://steampipe.io/docs/managing/connections#using-aggregators):

```hcl
connection "turbopuffer_prod" {
  plugin  = "somoore/turbopuffer"
  api_key = "tpuf_prod_..."
  regions = ["gcp-us-central1"]
}

connection "turbopuffer_staging" {
  plugin  = "somoore/turbopuffer"
  api_key = "tpuf_staging_..."
  regions = ["gcp-us-central1"]
}

connection "turbopuffer_all" {
  plugin      = "somoore/turbopuffer"
  type        = "aggregator"
  connections = ["turbopuffer_prod", "turbopuffer_staging"]
}
```

Querying `turbopuffer_all.turbopuffer_namespace` fans out across every member connection and adds a `_ctx` column identifying which one each row came from. See [Using Aggregators](https://steampipe.io/docs/managing/connections#using-aggregators).

## The control-plane gap (read this)

turbopuffer's public API is **data-plane only**. API keys, their permissions, organization membership and billing exist only in the dashboard — so there is deliberately no `turbopuffer_api_key` table. When that management API ships, key-hygiene tables land here first.
