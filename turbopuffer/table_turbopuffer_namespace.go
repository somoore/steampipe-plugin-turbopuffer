package turbopuffer

import (
	"context"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// namespaceRow is the streamed shape for turbopuffer_namespace.
type namespaceRow struct {
	ID     string
	Region string
}

func tableTurbopufferNamespace(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "turbopuffer_namespace",
		Description: "Namespaces in the turbopuffer organization, with size, activity, schema and encryption posture from the metadata API.",
		List: &plugin.ListConfig{
			Hydrate: listNamespaces,
			// ~~ is the Postgres LIKE operator; pushed down as ?prefix=.
			KeyColumns: []*plugin.KeyColumn{
				{Name: "id", Require: plugin.Optional, Operators: []string{"=", "~~"}},
			},
		},
		GetMatrixItemFunc: regionMatrix,
		HydrateConfig: []plugin.HydrateConfig{
			// One metadata GET per namespace; cap concurrency.
			{Func: getNamespaceMetadata, MaxConcurrency: 10},
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Namespace ID."},
			{Name: "approx_logical_bytes", Type: proto.ColumnType_INT, Hydrate: getNamespaceMetadata, Transform: transform.FromField("ApproxLogicalBytes"), Description: "Approximate logical size in bytes."},
			{Name: "approx_row_count", Type: proto.ColumnType_INT, Hydrate: getNamespaceMetadata, Transform: transform.FromField("ApproxRowCount"), Description: "Approximate number of rows."},
			{Name: "created_at", Type: proto.ColumnType_TIMESTAMP, Hydrate: getNamespaceMetadata, Transform: transform.FromField("CreatedAt"), Description: "When the namespace was created."},
			{Name: "encryption_key_name", Type: proto.ColumnType_STRING, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Encryption.KeyName"), Description: "CMEK key resource name, if customer-managed encryption is enabled."},
			{Name: "encryption_mode", Type: proto.ColumnType_STRING, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Encryption.Mode"), Description: "Encryption mode; 'customer_managed' when CMEK is configured."},
			{Name: "index_status", Type: proto.ColumnType_STRING, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Index.Status"), Description: "Index status; 'up-to-date' or 'updating' when writes are not yet fully indexed."},
			{Name: "index_unindexed_bytes", Type: proto.ColumnType_INT, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Index.UnindexedBytes"), Description: "Bytes written to the WAL but not yet indexed; nonzero means the index is lagging recent writes."},
			{Name: "pinning", Type: proto.ColumnType_JSON, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Pinning"), Description: "Pinning configuration and status, if any."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region hosting the namespace (e.g. gcp-us-central1)."},
			{Name: "schema", Type: proto.ColumnType_JSON, Hydrate: getNamespaceMetadata, Transform: transform.FromField("Schema"), Description: "Full attribute schema as JSON (attribute -> config)."},
			{Name: "updated_at", Type: proto.ColumnType_TIMESTAMP, Hydrate: getNamespaceMetadata, Transform: transform.FromField("UpdatedAt"), Description: "Last write to the namespace. The staleness signal."},
			{Name: "akas", Type: proto.ColumnType_JSON, Transform: transform.FromValue().Transform(namespaceAkas), Description: "Array of globally unique identifiers (region/id) for the namespace."},
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Title of the resource."},
		},
	}
}

//// LIST HYDRATE FUNCTIONS

// listNamespaces streams every namespace in the current region.
func listNamespaces(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := regionFromMatrix(ctx)
	client, err := getClient(ctx, d, region)
	if err != nil {
		return nil, err
	}

	params := tpuf.NamespacesParams{PageSize: tpuf.Int(500)}
	// Push id qual down as an API-side prefix filter.
	if q := d.EqualsQualString("id"); q != "" {
		params.Prefix = tpuf.String(q)
	} else if quals := d.Quals["id"]; quals != nil {
		for _, qual := range quals.Quals {
			if qual.Operator == "~~" {
				if p, ok := likeToPrefix(qual.Value.GetStringValue()); ok {
					params.Prefix = tpuf.String(p)
				}
			}
		}
	}

	page, err := client.Namespaces(ctx, params)
	for page != nil {
		if err != nil {
			return nil, err
		}
		for _, ns := range page.Namespaces {
			d.StreamListItem(ctx, namespaceRow{ID: ns.ID, Region: region})
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}
		page, err = page.GetNextPage()
	}
	return nil, err
}

//// HYDRATE FUNCTIONS

// getNamespaceMetadata hydrates a namespace row from the metadata API.
func getNamespaceMetadata(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	ns := h.Item.(namespaceRow)
	client, err := getClient(ctx, d, ns.Region)
	if err != nil {
		return nil, err
	}
	nsClient := client.Namespace(ns.ID)
	meta, err := nsClient.Metadata(ctx, tpuf.NamespaceMetadataParams{})
	if err != nil {
		plugin.Logger(ctx).Error("turbopuffer_namespace.getNamespaceMetadata", "namespace", ns.ID, "error", err)
		return nil, err
	}
	return meta, nil
}

// likeToPrefix turns 'abc%' into prefix "abc"; other patterns stay in Postgres.
func likeToPrefix(pattern string) (string, bool) {
	if pattern == "" {
		return "", false
	}
	body := pattern[:len(pattern)-1]
	if pattern[len(pattern)-1] == '%' && !containsWildcard(body) {
		return body, true
	}
	return "", false
}

func containsWildcard(s string) bool {
	for _, c := range s {
		if c == '%' || c == '_' {
			return true
		}
	}
	return false
}

//// TRANSFORM FUNCTIONS

// namespaceAkas builds the akas array: region/id.
func namespaceAkas(_ context.Context, td *transform.TransformData) (interface{}, error) {
	ns := td.HydrateItem.(namespaceRow)
	return []string{ns.Region + "/" + ns.ID}, nil
}
