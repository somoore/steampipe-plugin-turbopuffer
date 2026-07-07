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
