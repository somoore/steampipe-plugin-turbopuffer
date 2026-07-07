# Example Queries

A getting-started tour of the turbopuffer plugin. Install and configure per the
[main docs](index.md), then work top to bottom — each section builds on the last.
Run these in `steampipe query`, or paste them into any Postgres client connected
to Steampipe.

## Get oriented

### List every namespace you can see

The basic inventory: one row per namespace per region.

```sql
select
  id,
  region,
  approx_row_count,
  approx_logical_bytes,
  updated_at
from
  turbopuffer_namespace
order by
  id;
```

### Size up the whole estate

Totals across every namespace and region.

```sql
select
  count(*) as namespaces,
  sum(approx_row_count) as total_rows,
  pg_size_pretty(sum(approx_logical_bytes)) as total_size
from
  turbopuffer_namespace;
```

### Find your largest namespaces

Where the data (and the risk) concentrates.

```sql
select
  id,
  region,
  approx_row_count,
  pg_size_pretty(approx_logical_bytes) as size
from
  turbopuffer_namespace
order by
  approx_logical_bytes desc
limit 10;
```

## Check security posture

### Find namespaces without customer-managed encryption

Namespaces on provider-default encryption. Alarm or accept deliberately.

```sql
select
  id,
  region,
  encryption_mode,
  encryption_key_name
from
  turbopuffer_namespace
where
  coalesce(encryption_mode, '') <> 'customer_managed';
```

### Find stale namespaces

No writes in 90 days — candidates for review or deletion.

```sql
select
  id,
  region,
  updated_at
from
  turbopuffer_namespace
where
  updated_at < now() - interval '90 days'
order by
  updated_at;
```

### Catch namespaces whose index is lagging

Unindexed bytes mean recent writes are not yet searchable.

```sql
select
  id,
  region,
  index_status,
  index_unindexed_bytes
from
  turbopuffer_namespace
where
  index_status is distinct from 'up-to-date'
  or index_unindexed_bytes > 0;
```

### Flag namespaces that break your naming convention

Naming drift is usually the first sign of an unmanaged namespace.

```sql
select
  id,
  region
from
  turbopuffer_namespace
where
  id !~ '^[a-z][a-z0-9-]*$';
```

## Audit the ACL gap

turbopuffer has no row-level RBAC — tenant isolation happens in *your* query
filters. That only works if the attribute you filter on is actually
`filterable`. These three queries are the reason this plugin exists.

### Find ACL-like attributes that are not filterable

An ACL attribute that exists but is not filterable is silent false security:
the application *thinks* it is isolating tenants, and turbopuffer cannot
enforce it.

```sql
select
  namespace,
  region,
  name,
  type,
  filterable
from
  turbopuffer_namespace_attribute
where
  name ~* '(acl|tenant|group|role|permission|owner)'
  and not filterable;
```

### Find full-text attributes that lost filterability

BM25 full-text attributes are not filterable by default — easy to miss when a
field doubles as an access-control value.

```sql
select
  namespace,
  region,
  name,
  full_text_search,
  filterable
from
  turbopuffer_namespace_attribute
where
  full_text_search
  and not filterable;
```

### List namespaces with no filterable attributes at all

Nothing to filter on means no way to scope queries per tenant.

```sql
select
  n.id,
  n.region
from
  turbopuffer_namespace as n
  left join turbopuffer_namespace_attribute as a
    on a.namespace = n.id
    and a.region = n.region
    and a.filterable
group by
  n.id,
  n.region
having
  count(a.name) = 0;
```

## Inspect documents (canary checks)

`turbopuffer_document` requires a `namespace` qual, never returns more than 100
rows, and always excludes vectors — it is for targeted governance lookups, not
export.

### Look up a canary document

Verify a known sentinel document is (or is not) retrievable.

```sql
select
  id,
  attributes
from
  turbopuffer_document
where
  namespace = 'prod-acme'
  and id = 'canary-001';
```

### Sample a few documents from a namespace

Spot-check what attribute shapes actually live in a namespace.

```sql
select
  id,
  attributes
from
  turbopuffer_document
where
  namespace = 'prod-acme'
limit 5;
```

## Evaluate recall (costs real queries)

Each row runs live ANN searches **plus exhaustive ground-truth searches**
against the namespace — bound the cost with `queries` and `top_k` quals, and
never scan this table unqualified.

### Measure ANN recall for one namespace

1.0 means the ANN index perfectly matches exhaustive search.

```sql
select
  namespace,
  queries,
  top_k,
  avg_recall
from
  turbopuffer_namespace_recall
where
  namespace = 'prod-acme'
  and queries = 10
  and top_k = 10;
```

## Map your regions

### Show the regions this connection scans

Each region is a separate endpoint; unlisted regions are invisible to every
query above.

```sql
select
  region,
  endpoint
from
  turbopuffer_region;
```
