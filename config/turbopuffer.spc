connection "turbopuffer" {
  plugin = "somoore/turbopuffer"

  # API key (or set the TURBOPUFFER_API_KEY environment variable). Prefer a key
  # scoped to read-only if your org uses key permissions — this plugin only reads.
  # api_key = "tpuf_..."

  # Regions to scan. Each region is a separate turbopuffer endpoint
  # (https://<region>.turbopuffer.com); the plugin fans out across all of them
  # and tags every row with its region. List ALL regions your org can reach.
  regions = ["gcp-us-central1"]
}
