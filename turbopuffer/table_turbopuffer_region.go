package turbopuffer

import (
	"context"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// regionRow is a configured region and its endpoint (no list-regions API).
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
			{Name: "endpoint", Type: proto.ColumnType_STRING, Transform: transform.FromField("Endpoint"), Description: "Region API endpoint."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region (e.g. gcp-us-central1)."},
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "Title of the resource."},
		},
	}
}

//// LIST HYDRATE FUNCTIONS

// listRegions streams the connection's configured regions and their endpoints.
func listRegions(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	for _, r := range configuredRegions(GetConfig(d.Connection)) {
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
