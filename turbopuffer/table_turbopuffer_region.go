package turbopuffer

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// turbopuffer has no public list-regions endpoint, so this table reflects the
// regions configured on the connection — which is exactly what residency
// audits need: "these are the regions this org can reach". Keep the
// connection's `regions` list equal to ALL regions your API key can write to,
// or shadow regions stay invisible to every control.
type regionRow struct {
	Region   string
	Endpoint string
}

func tableTurbopufferRegion(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "turbopuffer_region",
		Description: "Regions configured on this Steampipe connection, with their API endpoints. Join anchor for residency queries.",
		List: &plugin.ListConfig{
			Hydrate: listRegions,
		},
		Columns: []*plugin.Column{
			// No List key columns, so all non-standard columns are alphabetical.
			{Name: "endpoint", Type: proto.ColumnType_STRING, Transform: transform.FromField("Endpoint"), Description: "Region API endpoint."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "Region identifier, e.g. gcp-us-central1."},

			// Steampipe standard column. Region is a config echo, not an API
			// resource, so akas/tags do not apply.
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "Title of the resource."},
		},
	}
}

//// LIST HYDRATE FUNCTIONS

// listRegions streams the connection's configured regions and their endpoints.
func listRegions(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	cfg := GetConfig(d.Connection)
	regions := cfg.Regions
	if len(regions) == 0 {
		regions = []string{defaultRegion}
	}
	for _, r := range regions {
		d.StreamListItem(ctx, regionRow{
			Region:   r,
			Endpoint: "https://" + r + ".turbopuffer.com",
		})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}
