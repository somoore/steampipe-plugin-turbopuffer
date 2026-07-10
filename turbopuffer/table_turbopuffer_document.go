package turbopuffer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbopuffer/turbopuffer-go/v2/packages/respjson"
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
		Description: "Targeted document lookups within a namespace (vectors always excluded). Requires a namespace qual; supports id equality pushdown. Scans are capped at 100 documents — intended for canary checks and small governance samples, not bulk export.",
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
			{Name: "attributes", Type: proto.ColumnType_JSON, Transform: transform.FromField("Attributes"), Description: "Document attributes as JSON. Vectors are never included; the document id is in the id column, not here."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region (e.g. gcp-us-central1)."},
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

	// Strip vector-typed attributes by schema, not just the "vector"
	// name, so custom-named embeddings never surface.
	meta, err := nsClient.Metadata(ctx, tpuf.NamespaceMetadataParams{})
	if err != nil {
		plugin.Logger(ctx).Error("turbopuffer_document.listDocuments", "namespace", namespace, "region", region, "error", err)
		return nil, err
	}
	vectorAttr := map[string]bool{}
	for name, attr := range meta.Schema {
		t := string(attr.Type)
		if strings.HasPrefix(t, "[") && (strings.Contains(t, "f16") || strings.Contains(t, "f32")) {
			vectorAttr[name] = true
		}
	}

	res, err := nsClient.Query(ctx, params)
	if err != nil {
		plugin.Logger(ctx).Error("turbopuffer_document.listDocuments", "namespace", namespace, "region", region, "error", err)
		return nil, err
	}

	// tpuf.Row is a map[string]any; `id` and `vector` are keys. The id is
	// excluded from attributes: it's already the id column, and a copy inside
	// the JSON would round-trip through float64 in the SDK's JSON pipeline,
	// corrupting integer ids above 2^53.
	for _, row := range res.Rows {
		attrs := make(map[string]interface{}, len(row))
		for k, v := range row {
			if k == "id" || k == "vector" || vectorAttr[k] {
				continue
			}
			attrs[k] = v
		}
		d.StreamListItem(ctx, documentRow{
			Namespace:  namespace,
			Region:     region,
			ID:         documentID(row["id"]),
			Attributes: attrs,
		})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}

// documentID renders a document id as text without numeric formatting
// artifacts. turbopuffer ids are string | int64; the SDK decodes untyped
// JSON numbers as respjson.Number (raw text preserved), but every numeric
// shape is handled explicitly so integer ids can never render in scientific
// notation or lose precision.
func documentID(v interface{}) string {
	switch id := v.(type) {
	case string:
		return id
	case respjson.Number:
		return string(id)
	case json.Number:
		return string(id)
	case int64:
		return strconv.FormatInt(id, 10)
	case float64:
		return strconv.FormatFloat(id, 'f', -1, 64)
	default:
		return fmt.Sprint(id)
	}
}
