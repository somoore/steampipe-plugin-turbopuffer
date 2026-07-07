# Table: turbopuffer_region

Regions configured on this connection, with endpoints. turbopuffer has no public list-regions API, so this reflects the connection config — keep `regions` equal to all regions the org can reach, or shadow regions stay invisible to every residency query.

## Examples

### Basic info

```sql
select
  region,
  endpoint
from
  turbopuffer_region;
```

### List regions configured but holding zero namespaces

```sql
select
  r.region
from
  turbopuffer_region r
  left join turbopuffer_namespace n on n.region = r.region
group by
  r.region
having
  count(n.id) = 0;
```

### Count namespaces and total logical size per region

```sql
select
  r.region,
  count(n.id) as namespaces,
  coalesce(sum(n.approx_logical_bytes), 0) as total_bytes
from
  turbopuffer_region r
  left join turbopuffer_namespace n on n.region = r.region
group by
  r.region
order by
  total_bytes desc;
```

### List namespaces hosted outside the approved EU regions

```sql
select
  n.id,
  n.region
from
  turbopuffer_namespace n
  join turbopuffer_region r on r.region = n.region
where
  n.id ~* '(^|[-_])eu([-_]|$)'
  and r.region !~* '(eu|europe)';
```
