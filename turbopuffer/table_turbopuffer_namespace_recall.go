package turbopuffer

import (
	"context"
	"fmt"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// recallRow reports how closely ANN search approximates exhaustive.
type recallRow struct {
	Namespace          string
	Region             string
	AvgRecall          float64
	AvgAnnCount        float64
	AvgExhaustiveCount float64
	Queries            int64
	TopK               int64
}

func tableTurbopufferNamespaceRecall(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "turbopuffer_namespace_recall",
		Description: "On-demand recall evaluation via POST /v1/namespaces/:ns/_debug/recall. COSTS REAL QUERIES: each row runs `num` ANN searches plus exhaustive ground-truth searches. Requires a namespace qual; never scanned automatically.",
		List: &plugin.ListConfig{
			Hydrate: listNamespaceRecall,
			KeyColumns: []*plugin.KeyColumn{
				{Name: "namespace", Require: plugin.Required},
				{Name: "queries", Require: plugin.Optional, CacheMatch: "exact"},
				{Name: "top_k", Require: plugin.Optional, CacheMatch: "exact"},
			},
		},
		GetMatrixItemFunc: regionMatrix,
		Columns: []*plugin.Column{
			{Name: "namespace", Type: proto.ColumnType_STRING, Transform: transform.FromField("Namespace"), Description: "Namespace ID (required qual)."},
			{Name: "queries", Type: proto.ColumnType_INT, Transform: transform.FromField("Queries"), Description: "Number of evaluation searches run (qual; server default if unset)."},
			{Name: "top_k", Type: proto.ColumnType_INT, Transform: transform.FromField("TopK"), Description: "Nearest neighbors evaluated per search (qual; server default if unset)."},
			{Name: "avg_ann_count", Type: proto.ColumnType_DOUBLE, Transform: transform.FromField("AvgAnnCount"), Description: "Average documents returned by the ANN searches."},
			{Name: "avg_exhaustive_count", Type: proto.ColumnType_DOUBLE, Transform: transform.FromField("AvgExhaustiveCount"), Description: "Average documents returned by the exhaustive searches."},
			{Name: "avg_recall", Type: proto.ColumnType_DOUBLE, Transform: transform.FromField("AvgRecall"), Description: "Average recall of ANN search vs exhaustive ground truth (1.0 = perfect)."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region."},

			// No akas: recall is an evaluation, not an API resource.
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("Namespace"), Description: "Title of the resource."},
		},
	}
}

//// LIST HYDRATE FUNCTIONS

// listNamespaceRecall runs one recall evaluation and streams a row.
func listNamespaceRecall(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := regionFromMatrix(ctx)
	namespace := d.EqualsQualString("namespace")
	if namespace == "" {
		return nil, fmt.Errorf("turbopuffer_namespace_recall requires a namespace qual, e.g. where namespace = 'prod-acme'")
	}

	client, err := getClient(ctx, d, region)
	if err != nil {
		return nil, err
	}

	params := tpuf.NamespaceRecallParams{}
	var queries, topK int64
	if q := d.EqualsQuals["queries"]; q != nil {
		queries = q.GetInt64Value()
		params.Num = tpuf.Int(queries)
	}
	if q := d.EqualsQuals["top_k"]; q != nil {
		topK = q.GetInt64Value()
		params.TopK = tpuf.Int(topK)
	}

	nsClient := client.Namespace(namespace)
	res, err := nsClient.Recall(ctx, params)
	if err != nil {
		return nil, err
	}

	d.StreamListItem(ctx, recallRow{
		Namespace:          namespace,
		Region:             region,
		AvgRecall:          res.AvgRecall,
		AvgAnnCount:        res.AvgAnnCount,
		AvgExhaustiveCount: res.AvgExhaustiveCount,
		Queries:            queries,
		TopK:               topK,
	})
	return nil, nil
}
