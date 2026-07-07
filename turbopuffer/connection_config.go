package turbopuffer

import (
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type turbopufferConfig struct {
	// Falls back to TURBOPUFFER_API_KEY. Use a read-only key if available.
	APIKey *string `hcl:"api_key"`

	// Regions to fan out across. Defaults to ["gcp-us-central1"].
	Regions []string `hcl:"regions,optional"`
}

// ConfigInstance returns an empty config for the SDK to populate.
func ConfigInstance() interface{} {
	return &turbopufferConfig{}
}

// GetConfig extracts the config from a connection, or a zero value.
func GetConfig(connection *plugin.Connection) turbopufferConfig {
	if connection == nil || connection.Config == nil {
		return turbopufferConfig{}
	}
	config, _ := connection.Config.(turbopufferConfig)
	return config
}
