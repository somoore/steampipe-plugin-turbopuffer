package turbopuffer

import (
	"context"
	"errors"
	"os"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbopuffer/turbopuffer-go/v2/option"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

// Endpoints used (verified against the official Go client, turbopuffer-go/v2):
//
//	GET  /v1/namespaces                     -> paginated {id} summaries
//	GET  /v2/namespaces/:ns/metadata        -> approx_row_count, approx_logical_bytes,
//	                                           created_at, updated_at,
//	                                           encryption{mode,key_name},
//	                                           schema{attr -> config}, sharding, pinning
//	POST /v2/namespaces/:ns/query           -> rows (used by turbopuffer_document)
//
// Base URL pattern: https://<region>.turbopuffer.com/
const defaultRegion = "gcp-us-central1"

//// CLIENT

// getClient returns a region-scoped turbopuffer client, cached per
// connection+region so concurrent hydrates share transports.
func getClient(ctx context.Context, d *plugin.QueryData, region string) (*tpuf.Client, error) {
	cacheKey := "turbopuffer-client-" + region
	if cached, ok := d.ConnectionManager.Cache.Get(cacheKey); ok {
		return cached.(*tpuf.Client), nil
	}

	cfg := GetConfig(d.Connection)
	apiKey := os.Getenv("TURBOPUFFER_API_KEY")
	if cfg.APIKey != nil {
		apiKey = *cfg.APIKey
	}
	if apiKey == "" {
		return nil, errors.New("turbopuffer: api_key must be set in the connection config or TURBOPUFFER_API_KEY env var")
	}

	client := tpuf.NewClient(
		option.WithAPIKey(apiKey),
		option.WithRegion(region), // substituted into https://REGION.turbopuffer.com/
	)

	d.ConnectionManager.Cache.Set(cacheKey, &client)
	return &client, nil
}

//// ERROR HANDLING

// isNotFoundError treats 404s as ignorable: a namespace deleted between the
// list call and its metadata hydrate should drop the row, not fail the scan.
func isNotFoundError(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, err error) bool {
	var apierr *tpuf.Error
	if errors.As(err, &apierr) {
		return apierr.StatusCode == 404
	}
	return false
}

// shouldRetryError backs off on rate limits and transient server errors.
func shouldRetryError(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData, err error) bool {
	var apierr *tpuf.Error
	if errors.As(err, &apierr) {
		return apierr.StatusCode == 429 || apierr.StatusCode >= 500
	}
	return false
}
