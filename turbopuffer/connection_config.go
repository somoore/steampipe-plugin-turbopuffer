package turbopuffer

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type turbopufferConfig struct {
	// API key (starts with tpuf_). Falls back to TURBOPUFFER_API_KEY env var.
	// SECURITY: use a read-only key if your org has one; this plugin only
	// ever issues GET /namespaces, GET /metadata and POST /query calls.
	APIKey *string `hcl:"api_key"`

	// Regions to scan, e.g. ["gcp-us-central1", "aws-eu-central-1"].
	// Each region is a separate turbopuffer base URL; the plugin fans out
	// across all of them. Defaults to ["gcp-us-central1"].
	Regions []string `hcl:"regions,optional"`
}

// ConfigInstance returns a new, empty turbopufferConfig for the SDK to
// populate from the connection's HCL.
func ConfigInstance() interface{} {
	return &turbopufferConfig{}
}

// GetConfig extracts the turbopufferConfig from a connection, returning a
// zero-value config if none is set.
func GetConfig(connection *plugin.Connection) turbopufferConfig {
	if connection == nil || connection.Config == nil {
		return turbopufferConfig{}
	}
	config, _ := connection.Config.(turbopufferConfig)
	return config
}
