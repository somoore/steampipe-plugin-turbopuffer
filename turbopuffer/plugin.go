// Package turbopuffer is a read-only Steampipe plugin exposing a
// turbopuffer organization's data-plane API as SQL tables.
package turbopuffer

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// Plugin returns the turbopuffer plugin definition.
func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name: "steampipe-plugin-turbopuffer",
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: ConfigInstance,
		},
		DefaultTransform: transform.FromCamel(), // no NullIfZero: keep filterable=false
		DefaultIgnoreConfig: &plugin.IgnoreConfig{
			ShouldIgnoreErrorFunc: isNotFoundError,
		},
		DefaultRetryConfig: &plugin.RetryConfig{
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

// regionMatrix builds one matrix item per configured region.
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
