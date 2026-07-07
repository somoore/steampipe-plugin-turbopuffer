// Package turbopuffer implements a Steampipe plugin that exposes a
// turbopuffer organization's data-plane API as SQL tables: namespaces, their
// schema attributes, documents, recall evaluations, and configured regions.
// It is read-only and region-scoped — each configured region is a separate
// base URL, fanned out via a query matrix.
package turbopuffer

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// Plugin returns the turbopuffer plugin definition.
//
// Design notes:
//   - turbopuffer is region-scoped: each region is a separate base URL
//     (https://<region>.turbopuffer.com). We model this with a query matrix,
//     so tables fan out across all configured regions in parallel and carry
//     a `region` column — the same pattern the AWS plugin uses.
//   - The public API is data-plane only. API keys, org membership, and
//     billing are dashboard-only today, which is why there is no
//     `turbopuffer_api_key` table (see docs/index.md: "control-plane gap").
//   - DefaultTransform is FromCamel WITHOUT NullIfZero: `filterable = false`
//     is a security-critical value we must never null out.
func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name: "steampipe-plugin-turbopuffer",
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: ConfigInstance,
		},
		DefaultTransform: transform.FromCamel(),
		DefaultIgnoreConfig: &plugin.IgnoreConfig{
			// Namespaces can vanish between list and hydrate; that's a skip,
			// not a failure.
			ShouldIgnoreErrorFunc: isNotFoundError,
		},
		DefaultRetryConfig: &plugin.RetryConfig{
			// Back off politely on 429/5xx.
			ShouldRetryErrorFunc: shouldRetryError,
		},
		TableMap: map[string]*plugin.Table{
			"turbopuffer_namespace":           tableTurbopufferNamespace(ctx),
			"turbopuffer_namespace_attribute": tableTurbopufferNamespaceAttribute(ctx),
			"turbopuffer_namespace_recall":    tableTurbopufferNamespaceRecall(ctx),
			"turbopuffer_document":            tableTurbopufferDocument(ctx),
			"turbopuffer_region":              tableTurbopufferRegion(ctx),
		},
	}
	return p
}

const matrixKeyRegion = "region"

// regionMatrix builds one matrix item per configured region so that list/get
// hydrates run once per region and rows are tagged with their region.
func regionMatrix(ctx context.Context, d *plugin.QueryData) []map[string]interface{} {
	cfg := GetConfig(d.Connection)
	regions := cfg.Regions
	if len(regions) == 0 {
		regions = []string{defaultRegion}
	}
	matrix := make([]map[string]interface{}, 0, len(regions))
	for _, r := range regions {
		matrix = append(matrix, map[string]interface{}{matrixKeyRegion: r})
	}
	return matrix
}

// regionFromMatrix extracts the current region for a hydrate call.
func regionFromMatrix(ctx context.Context) string {
	if m := plugin.GetMatrixItem(ctx); m != nil {
		if r, ok := m[matrixKeyRegion].(string); ok {
			return r
		}
	}
	return defaultRegion
}
