package turbopuffer

import (
	"context"
	"fmt"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// documentRow is a thin projection of a queried row: id + attributes.
// Vectors are always excluded — this table exists for governance lookups
// (canary checks, targeted sampling), not for bulk export. Embeddings can be
// inverted to recover source text, so we deliberately never surface them.
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
			// Key columns first.
			{Name: "namespace", Type: proto.ColumnType_STRING, Transform: transform.FromField("Namespace"), Description: "Namespace ID (required qual)."},
			{Name: "id", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Document ID."},

			// Remaining columns, alphabetical.
			{Name: "attributes", Type: proto.ColumnType_JSON, Transform: transform.FromField("Attributes"), Description: "Document attributes as JSON. Vectors are never included."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region."},

			// Steampipe standard columns last.
			{Name: "akas", Type: proto.ColumnType_JSON, Transform: transform.FromValue().Transform(documentAkas), Description: "Array of globally unique identifiers (region/namespace/id) for the document."},
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID"), Description: "Title of the resource."},
		},
	}
}

// sampleCap bounds how many documents a single scan will pull when no id qual
// or LIMIT is supplied. Governance sampling should be cheap and polite.
const sampleCap int64 = 100

//// LIST HYDRATE FUNCTIONS

// listDocuments runs a bounded, deterministic query within one namespace and
// streams id + attributes, always stripping the vector.
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
		TopK: tpuf.Int(topK),
		// Ascending id order = deterministic, index-friendly scan.
		RankBy: tpuf.NewRankByAttribute("id", tpuf.RankByAttributeOrderAsc),
		// Include all attributes; we strip the vector in Go below.
		// (The API rejects include_attributes + exclude_attributes together.)
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

	// tpuf.Row is a map[string]any: `id` is a key, vectors live under "vector".
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

// documentAkas builds the standard akas array: region/namespace/id.
func documentAkas(_ context.Context, td *transform.TransformData) (interface{}, error) {
	r := td.HydrateItem.(documentRow)
	return []string{r.Region + "/" + r.Namespace + "/" + r.ID}, nil
}
