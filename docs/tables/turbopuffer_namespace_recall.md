# Table: turbopuffer_namespace_recall

On-demand recall evaluation: how closely ANN search approximates exhaustive ground truth. Silent recall degradation means retrieval is quietly wrong — an integrity signal worth watching.

> **Cost warning:** each row runs real ANN searches *plus* exhaustive searches in the namespace. Always qual `namespace`; never wildcard this table in scheduled scans without intent.

## Examples

### Basic info

```sql
select
  namespace,
  avg_recall,
  avg_ann_count,
  avg_exhaustive_count
from
  turbopuffer_namespace_recall
where
  namespace = 'prod-acme';
```

### Evaluate recall with 50 searches at top_k 10

```sql
select
  namespace,
  avg_recall
from
  turbopuffer_namespace_recall
where
  namespace = 'prod-acme'
  and queries = 50
  and top_k = 10;
```

### Flag a namespace whose recall dips below 0.9

```sql
select
  namespace,
  avg_recall,
  case
    when avg_recall < 0.9 then 'DEGRADED'
    else 'ok'
  end as verdict
from
  turbopuffer_namespace_recall
where
  namespace = 'prod-acme';
```

### List namespaces where the ANN index returns fewer results than exhaustive search

```sql
select
  namespace,
  avg_ann_count,
  avg_exhaustive_count,
  avg_exhaustive_count - avg_ann_count as shortfall
from
  turbopuffer_namespace_recall
where
  namespace = 'prod-acme'
  and avg_ann_count < avg_exhaustive_count;
```
