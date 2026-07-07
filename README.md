# turbopuffer plugin for Steampipe

Use SQL to query namespaces, schemas, documents and recall in [turbopuffer][].

- **[Get started →](docs/index.md)**
- Documentation: [Table definitions & examples](docs/tables)

```sql
select id, region, approx_row_count, encryption_mode, updated_at
from turbopuffer_namespace
order by approx_logical_bytes desc;
```

Pairs with the [turbopuffer Security Benchmark mod][mod] — 15 Powerpipe controls and branded dashboards over these tables.

## Quick start

Install the plugin with [Steampipe][]:

    steampipe plugin install somoore/turbopuffer

Configure your [connection](docs/index.md) with a turbopuffer API key and the regions to scan.

## Development

To build the plugin and install it in your `.steampipe` directory:

    make install

Copy the default config file:

    cp config/turbopuffer.spc ~/.steampipe/config/turbopuffer.spc

Run the tests and standards checks:

    make test

Built on the official [`turbopuffer-go`][sdk] client; endpoints and response fields were verified against its v2 surface. Unofficial community plugin — not affiliated with turbopuffer inc.

## License

Apache 2

[steampipe]: https://steampipe.io
[turbopuffer]: https://turbopuffer.com
[sdk]: https://github.com/turbopuffer/turbopuffer-go
[mod]: https://github.com/somoore/powerpipe-mod-turbopuffer-compliance
