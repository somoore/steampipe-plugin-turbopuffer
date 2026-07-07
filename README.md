# turbopuffer plugin for Steampipe

Query [turbopuffer](https://turbopuffer.com) — namespaces, schemas, documents, recall — with SQL.

- **[Table docs & examples](./docs/tables)** · **[Get started](./docs/index.md)**
- Pairs with the [turbopuffer Security Benchmark mod](../powerpipe-mod-turbopuffer-compliance) (15 Powerpipe controls + branded dashboards).

```sql
select namespace, name
from turbopuffer_namespace_attribute
where name in ('tenant_id') and not filterable;   -- isolation silently unenforceable
```

## Developing

```bash
git clone <this repo> && cd steampipe-plugin-turbopuffer
go mod tidy
make install          # builds into ~/.steampipe/plugins/local/turbopuffer/
cp config/turbopuffer.spc ~/.steampipe/config/
steampipe query "select * from turbopuffer_region"
```

Built on the official [`turbopuffer-go`](https://github.com/turbopuffer/turbopuffer-go) client; endpoints and response fields were verified against its v2 surface. Unofficial community plugin — not affiliated with turbopuffer inc. MIT licensed (switch to Apache-2.0 before any Hub submission if preferred; both are conventional).
