package turbopuffer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbopuffer/turbopuffer-go/v2/option"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

const defaultRegion = "gcp-us-central1"

// regionPattern is the permissive shape of a turbopuffer region id: a single
// lowercase DNS label. That is all the API uses (gcp-us-central1,
// aws-us-east-1); anything containing '/', '@', '?', '#', '.' or a space could
// rewrite the host in https://REGION.turbopuffer.com/ and send the bearer key
// somewhere else, so it is rejected.
var regionPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// validRegion reports whether r is safe to substitute into a region endpoint.
func validRegion(r string) bool {
	return len(r) <= 63 && regionPattern.MatchString(r)
}

//// CLIENT

// getClient returns a region-scoped client, cached per region.
func getClient(ctx context.Context, d *plugin.QueryData, region string) (*tpuf.Client, error) {
	// Defence in depth: never let a region from any source rewrite the base URL.
	if !validRegion(region) {
		return nil, fmt.Errorf("turbopuffer: invalid region %q: must be a lowercase DNS label matching %s", region, regionPattern)
	}

	cacheKey := "turbopuffer-client-" + region
	if cached, ok := d.ConnectionCache.Get(ctx, cacheKey); ok {
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
		option.WithRegion(region),
	)

	if err := d.ConnectionCache.Set(ctx, cacheKey, &client); err != nil {
		plugin.Logger(ctx).Warn("getClient", "region", region, "cache_set_error", err)
	}
	return &client, nil
}

// configuredRegions returns the connection's regions: trimmed, deduped,
// validated, and defaulted when unset. Invalid entries are dropped so a config
// value can never change the API host the key is sent to.
func configuredRegions(cfg turbopufferConfig) []string {
	regions := make([]string, 0, len(cfg.Regions))
	seen := map[string]bool{}
	for _, r := range cfg.Regions {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] || !validRegion(r) {
			continue
		}
		seen[r] = true
		regions = append(regions, r)
	}
	if len(regions) == 0 {
		return []string{defaultRegion}
	}
	return regions
}

// rejectedRegions returns configured regions that failed validation, so callers
// can say why a region disappeared instead of silently narrowing the matrix.
func rejectedRegions(cfg turbopufferConfig) []string {
	var rejected []string
	for _, r := range cfg.Regions {
		r = strings.TrimSpace(r)
		if r != "" && !validRegion(r) {
			rejected = append(rejected, r)
		}
	}
	return rejected
}

//// ERROR HANDLING

// isNotFoundError reports whether err is a 404 (an ignorable skip).
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
