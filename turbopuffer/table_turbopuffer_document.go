package turbopuffer

import (
	"context"
	"fmt"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// documentRow is a queried row: id + attributes, vector always stripped.
type documentRow struct {
	Namespace  string
	Region     string
	ID         string
	Attributes interface{}
}

func tableTurbopufferDocument(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "turbopuffer_document",
		Description: "Targeted document lookups within a namespace (vectors always excluded). Requires a namespace qual; supports id equality pushdown. Intended for canary checks and small governance samples, not bulk export.",
		List: &plugin.ListConfig{
			Hydrate: listDocuments,
			KeyColumns: []*plugin.KeyColumn{
				{Name: "namespace", Require: plugin.Required},
				{Name: "id", Require: plugin.Optional},
			},
		},
		GetMatrixItemFunc: regionMatrix,
		Columns: []*plugin.Column{
			{Name: "namespace", Type: proto.ColumnType_STRING, Transform: transform.FromField("Namespace"), Description: "Namespace ID (required qual)."},
			{Name: "id", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Document ID."},
			{Name: "attributes", Type: proto.ColumnType_JSON, Transform: transform.FromField("Attributes"), Description: "Document attributes as JSON. Vectors are never included."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region."},
			{Name: "akas", Type: proto.ColumnType_JSON, Transform: transform.FromValue().Transform(documentAkas), Description: "Array of globally unique identifiers (region/namespace/id) for the document."},
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Title of the resource."},
		},
	}
}

// sampleCap bounds a scan when no id qual or LIMIT is supplied.
const sampleCap int64 = 100

//// LIST HYDRATE FUNCTIONS

// listDocuments runs a bounded query within one namespace.
func listDocuments(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := regionFromMatrix(ctx)
	namespace := d.EqualsQualString("namespace")
	if namespace == "" {
		return nil, fmt.Errorf("turbopuffer_document requires a namespace qual, e.g. where namespace = 'prod-acme'")
	}

	client, err := getClient(ctx, d, region)
	if err != nil {
		return nil, err
	}

	topK := sampleCap
	if d.QueryContext.Limit != nil && *d.QueryContext.Limit < topK {
		topK = *d.QueryContext.Limit
	}

	params := tpuf.NamespaceQueryParams{
		TopK:   tpuf.Int(topK),
		RankBy: tpuf.NewRankByAttribute("id", tpuf.RankByAttributeOrderAsc), // deterministic scan
		// Include all; vector stripped in Go (API rejects include+exclude).
		IncludeAttributes: tpuf.IncludeAttributesParam{Bool: tpuf.Bool(true)},
	}
	if id := d.EqualsQualString("id"); id != "" {
		params.Filters = tpuf.NewFilterEq("id", id)
		params.TopK = tpuf.Int(1)
	}

	nsClient := client.Namespace(namespace)
	res, err := nsClient.Query(ctx, params)
	if err != nil {
		return nil, err
	}

	// tpuf.Row is a map[string]any; `id` and `vector` are keys.
	for _, row := range res.Rows {
		attrs := make(map[string]interface{}, len(row))
		for k, v := range row {
			attrs[k] = v
		}
		delete(attrs, "vector")
		d.StreamListItem(ctx, documentRow{
			Namespace:  namespace,
			Region:     region,
			ID:         fmt.Sprint(row["id"]),
			Attributes: attrs,
		})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}

//// TRANSFORM FUNCTIONS

// documentAkas builds the akas array: region/namespace/id.
func documentAkas(_ context.Context, td *transform.TransformData) (interface{}, error) {
	r := td.HydrateItem.(documentRow)
	return []string{r.Region + "/" + r.Namespace + "/" + r.ID}, nil
}
