package main

import (
	"github.com/somoore/steampipe-plugin-turbopuffer/turbopuffer"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{PluginFunc: turbopuffer.Plugin})
}
