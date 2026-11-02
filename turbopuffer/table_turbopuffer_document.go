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

	topK := effectiveTopK(d.QueryContext.Limit)

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
		if isVectorType(string(attr.Type)) {
			vectorAttr[name] = true
		}
	}

	res, err := nsClient.Query(ctx, params)
	if err != nil {
		plugin.Logger(ctx).Error("turbopuffer_document.listDocuments", "namespace", namespace, "region", region, "error", err)
		return nil, err
	}

	for _, row := range res.Rows {
		d.StreamListItem(ctx, documentRow{
			Namespace:  namespace,
			Region:     region,
			ID:         documentID(row["id"]),
			Attributes: documentAttributes(row, vectorAttr),
		})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}

// effectiveTopK bounds a scan to sampleCap; a non-positive LIMIT is ignored
// rather than forwarded as top_k=0, which the API rejects.
func effectiveTopK(limit *int64) int64 {
	if limit != nil && *limit > 0 && *limit < sampleCap {
		return *limit
	}
	return sampleCap
}

// isVectorType reports whether a schema type is an embedding and must never be
// returned in the attributes column. Dense vectors are [DIMS]f16/f32 (and the
// multi-vector []DIMS forms) and sparse vectors are {}f16. Scalar array types
// such as []float and []string are legitimate filter values, not vectors.
func isVectorType(t string) bool {
	if !strings.HasPrefix(t, "[") && !strings.HasPrefix(t, "{}") {
		return false
	}
	// The element type follows the final dimension/brace marker:
	// "[8]f32" -> "f32", "[][8]f32" -> "f32", "{}f16" -> "f16".
	elem := t[strings.LastIndexAny(t, "]}")+1:]
	switch elem {
	case "f16", "f32", "f64", "bf16":
		return true
	}
	return false
}

// documentAttributes copies a row without the id, the default vector, or any
// schema-identified vector attribute. The id is already the id column, and a
// copy inside the JSON would round-trip through float64 in the SDK's JSON
// pipeline, corrupting integer ids above 2^53.
func documentAttributes(row tpuf.Row, vectorAttr map[string]bool) map[string]interface{} {
	attrs := make(map[string]interface{}, len(row))
	for k, v := range row {
		if k == "id" || k == "vector" || vectorAttr[k] {
			continue
		}
		attrs[k] = v
	}
	return attrs
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
