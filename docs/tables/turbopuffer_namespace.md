# Table: turbopuffer_namespace

Namespaces in the organization, one row per namespace per configured region, hydrated from the metadata API: size, activity, encryption, and full schema.

## Examples

### Basic info

```sql
select
  id,
  region,
  approx_row_count,
  created_at
from
  turbopuffer_namespace;
```

### List production namespaces without customer-managed encryption

```sql
select
  id,
  region
from
  turbopuffer_namespace
where
  id like 'prod-%'
  and coalesce(encryption_key_name, '') = '';
```

### List stale namespaces with no writes in 90 days

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

### List namespaces over 250 GB for blast-radius review

```sql
select
  id,
  region,
  round(approx_logical_bytes / 1073741824.0, 1) as gb
from
  turbopuffer_namespace
where
  approx_logical_bytes > 250 * 1073741824::bigint
order by
  approx_logical_bytes desc;
```
