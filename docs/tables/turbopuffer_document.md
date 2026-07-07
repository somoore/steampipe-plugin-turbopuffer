# Table: turbopuffer_document

Targeted document lookups within one namespace. **Requires a `namespace` qual**; supports `id` equality pushdown. Vectors are never returned — embeddings are invertible, and this table exists for governance lookups (canaries, small samples), not export.

## Examples

### Basic info

```sql
select
  namespace,
  id,
  attributes
from
  turbopuffer_document
where
  namespace = 'prod-acme';
```

### Get a canary honeytoken document by id

```sql
select
  namespace,
  id
from
  turbopuffer_document
where
  namespace = 'prod-acme'
  and id = 'canary-00000000';
```

### List a sample of documents for classification review

```sql
select
  id,
  attributes
from
  turbopuffer_document
where
  namespace = 'prod-acme'
limit 20;
```

### List canary coverage across tenant namespaces

```sql
select
  n.id as namespace,
  (d.id is not null) as canary_present
from
  turbopuffer_namespace n
  left join turbopuffer_document d on d.namespace = n.id
  and d.region = n.region
  and d.id = 'canary-00000000'
where
  n.id like 'prod-%';
```
