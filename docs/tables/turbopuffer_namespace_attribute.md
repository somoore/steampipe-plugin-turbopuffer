# Table: turbopuffer_namespace_attribute

One row per attribute in each namespace schema. This is the tenant-isolation preconditions table: turbopuffer has no built-in row-level RBAC, so isolation is only as strong as the filters applied on these attributes.

## Examples

### Basic info

```sql
select
  namespace,
  name,
  type,
  filterable
from
  turbopuffer_namespace_attribute;
```

### List ACL attributes that exist but are not filterable

```sql
select
  namespace,
  region,
  name
from
  turbopuffer_namespace_attribute
where
  name in ('tenant_id', 'user_id', 'group_ids')
  and not filterable;
```

### List sensitive-named attributes stored next to vectors

```sql
select
  namespace,
  name,
  type
from
  turbopuffer_namespace_attribute
where
  name ~* '(ssn|credit_card|password|secret|token|api_key|medical)';
```

### List search-amplified sensitive fields

```sql
select
  namespace,
  name,
  full_text_search,
  regex,
  glob,
  fuzzy
from
  turbopuffer_namespace_attribute
where
  name ~* '(ssn|credit_card|password|secret|token)'
  and (full_text_search or regex or glob or fuzzy);
```
